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

from cortex.lib.exceptions import UserException, CortexException
from cortex.lib.log import get_logger

logger = get_logger()


def get_classes(ctx, api_name):
    api = ctx.apis[api_name]
    prefix = os.path.join(ctx.metadata_root, api["id"], "classes")
    class_paths = ctx.storage.search(prefix=prefix)
    class_set = set()
    for class_path in class_paths:
        encoded_class_name = class_path.split("/")[-1]
        class_set.add(base64.urlsafe_b64decode(encoded_class_name.encode()).decode())
    return class_set


def upload_class(ctx, api_name, class_name):
    api = ctx.apis[api_name]

    try:
        ascii_encoded = class_name.encode("ascii")  # cloudwatch only supports ascii
        encoded_class_name = base64.urlsafe_b64encode(ascii_encoded)
        key = os.path.join(ctx.metadata_root, api["id"], "classes", encoded_class_name.decode())
        ctx.storage.put_json("", key)
    except Exception as e:
        raise ValueError("unable to store class {}".format(class_name)) from e


def api_metric_dimensions(ctx, api_name):
    api = ctx.apis[api_name]
    return [
        {"Name": "AppName", "Value": ctx.app["name"]},
        {"Name": "APIName", "Value": api["name"]},
        {"Name": "APIID", "Value": api["id"]},
    ]


def status_code_metric(dimensions, status_code):
    status_code_series = int(status_code / 100)
    status_code_dimensions = dimensions + [
        {"Name": "Code", "Value": "{}XX".format(status_code_series)}
    ]
    return [{"MetricName": "StatusCode", "Dimensions": status_code_dimensions, "Value": 1}]


def extract_prediction(api, prediction):
    tracker = api.get("tracker")
    if tracker.get("key") is not None:
        key = tracker["key"]
        if type(prediction) != dict:
            raise ValueError(
                "failed to track key '{}': expected prediction to be of type dict but found '{}'".format(
                    key, type(prediction)
                )
            )
        if prediction.get(key) is None:
            raise ValueError(
                "failed to track key '{}': not found in prediction".format(tracker["key"])
            )
        predicted_value = prediction[key]
    else:
        predicted_value = prediction

    if tracker["model_type"] == "classification":
        if type(predicted_value) != str and type(predicted_value) != int:
            raise ValueError(
                "failed to track classification prediction: expected type 'str' or 'int' but encountered '{}'".format(
                    type(predicted_value)
                )
            )
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
    tracker = api.get("tracker")
    if tracker["model_type"] == "classification":
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


def cache_classes(ctx, api, prediction, class_set):
    if prediction not in class_set:
        upload_class(ctx, api["name"], prediction)
        logger.info(class_set)
        class_set.add(prediction)


def post_request_metrics(ctx, api, response, prediction_payload, class_set):
    api_name = api["name"]
    api_dimensions = api_metric_dimensions(ctx, api_name)
    metrics_list = []
    metrics_list += status_code_metric(api_dimensions, response.status_code)

    if prediction_payload is not None:
        if api.get("tracker") is not None:
            try:
                prediction = extract_prediction(api, prediction_payload)

                if api["tracker"]["model_type"] == "classification":
                    cache_classes(ctx, api, prediction, class_set)

                metrics_list += prediction_metrics(api_dimensions, api, prediction)
            except Exception as e:
                logger.warn(str(e), exc_info=True)

    try:
        logger.info(metrics_list)
        ctx.publish_metrics(metrics_list)
    except Exception as e:
        logger.warn(str(e), exc_info=True)
