# Copyright 2019 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import base64
import time

from cortex.lib.exceptions import UserException, CortexException
from cortex.lib.log import cx_logger


def get_classes(api):
    prefix = os.path.join(api.metadata_root, api.id, "classes")
    class_paths = api.storage.search(prefix=prefix)
    class_set = set()
    for class_path in class_paths:
        encoded_class_name = class_path.split("/")[-1]
        class_set.add(base64.urlsafe_b64decode(encoded_class_name.encode()).decode())
    return class_set


def upload_class(api, class_name):
    try:
        ascii_encoded = class_name.encode("ascii")  # cloudwatch only supports ascii
        encoded_class_name = base64.urlsafe_b64encode(ascii_encoded)
        key = os.path.join(api.metadata_root, api.id, "classes", encoded_class_name.decode())
        api.storage.put_json("", key)
    except Exception as e:
        raise ValueError("unable to store class {}".format(class_name)) from e


def api_metric_dimensions(api):
    return [{"Name": "APIName", "Value": api.name}, {"Name": "APIID", "Value": api.id}]


def status_code_metric(dimensions, status_code):
    status_code_series = int(status_code / 100)
    status_code_dimensions = dimensions + [
        {"Name": "Code", "Value": "{}XX".format(status_code_series)}
    ]
    return [
        {
            "MetricName": "StatusCode",
            "Dimensions": status_code_dimensions,
            "Value": 1,
            "Unit": "Count",
        }
    ]


def latency_metric(dimensions, start_time):
    return [
        {
            "MetricName": "Latency",
            "Dimensions": dimensions,
            "Value": (time.time() - start_time) * 1000,  # milliseconds
        }
    ]


def extract_prediction(api, prediction):
    if api.tracker.key is not None:
        if type(prediction) != dict:
            raise ValueError(
                "failed to track key '{}': expected prediction to be of type dict but found '{}'".format(
                    api.tracker.key, type(prediction)
                )
            )
        if prediction.get(api.tracker.key) is None:
            raise ValueError(
                "failed to track key '{}': not found in prediction".format(api.tracker.key)
            )
        predicted_value = prediction[api.tracker.key]
    else:
        predicted_value = prediction

    if api.tracker.model_type == "classification":
        if type(predicted_value) != str and type(predicted_value) != int:
            raise ValueError(
                "failed to track classification prediction: expected type 'str' or 'int' but encountered '{}'".format(
                    type(predicted_value)
                )
            )
        return str(predicted_value)
    else:
        if type(predicted_value) != float and type(predicted_value) != int:  # allow ints
            raise ValueError(
                "failed to track regression prediction: expected type 'float' or 'int' but encountered '{}'".format(
                    type(predicted_value)
                )
            )
    return predicted_value


def prediction_metrics(dimensions, api, prediction):
    metric_list = []
    if api.tracker.model_type == "classification":
        dimensions_with_class = dimensions + [{"Name": "Class", "Value": str(prediction)}]
        metric = {
            "MetricName": "Prediction",
            "Dimensions": dimensions_with_class,
            "Unit": "Count",
            "Value": 1,
        }

        metric_list.append(metric)
    else:
        metric = {"MetricName": "Prediction", "Dimensions": dimensions, "Value": float(prediction)}
        metric_list.append(metric)
    return metric_list


def cache_classes(api, prediction, class_set):
    if prediction not in class_set:
        upload_class(api.name, prediction)
        class_set.add(prediction)


def post_request_metrics(api, response, prediction_payload, start_time, class_set):
    api_dimensions = api_metric_dimensions(api)
    metrics_list = []
    metrics_list += status_code_metric(api_dimensions, response.status_code)

    if prediction_payload is not None:
        if api.tracker is not None:
            try:
                prediction = extract_prediction(api, prediction_payload)

                if api.tracker.model_type == "classification":
                    cache_classes(api, prediction, class_set)

                metrics_list += prediction_metrics(api_dimensions, api, prediction)
            except Exception as e:
                cx_logger().warn("unable to record prediction metric", exc_info=True)

    metrics_list += latency_metric(api_dimensions, start_time)
    try:
        if api.statsd is None:
            raise CortexException("statsd client not initialized")  # unexpected

        for metric in metrics_list:
            tags = ["{}:{}".format(dim["Name"], dim["Value"]) for dim in metric["Dimensions"]]
            if metric.get("Unit") == "Count":
                api.statsd.increment(metric["MetricName"], value=metric["Value"], tags=tags)
            else:
                api.statsd.histogram(metric["MetricName"], value=metric["Value"], tags=tags)
    except Exception as e:
        cx_logger().warn("failure encountered while publishing metrics", exc_info=True)
