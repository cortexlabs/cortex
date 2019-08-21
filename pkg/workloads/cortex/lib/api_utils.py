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
    prefix = os.path.join(api["metadata_key"], "classes")
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
        key = os.path.join(api["metadata_key"], "classes", encoded_class_name.decode())
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


def predictions_per_request_metric(dimensions, prediction_count):
    return [
        {"MetricName": "PredictionsPerRequest", "Dimensions": dimensions, "Value": prediction_count}
    ]


def extract_predicted_values(api, predictions):
    predicted_values = []

    tracker = api.get("tracker")
    for prediction in predictions:
        predicted_value = prediction.get(tracker["key"])
        if predicted_value is None:
            raise ValueError(
                "failed to track key '{}': not found in response payload".format(tracker["key"])
            )
        if tracker["model_type"] == "classification":
            if type(predicted_value) != str and type(predicted_value) != int:
                raise ValueError(
                    "failed to track key '{}': expected type 'str' or 'int' but encountered '{}'".format(
                        tracker["key"], type(predicted_value)
                    )
                )
        else:
            if type(predicted_value) != float and type(predicted_value) != int:  # allow ints
                raise ValueError(
                    "failed to track key '{}': expected type 'float' or 'int' but encountered '{}'".format(
                        tracker["key"], type(predicted_value)
                    )
                )
        predicted_values.append(predicted_value)

    return predicted_values


def prediction_metrics(dimensions, api, predicted_values):
    metric_list = []
    tracker = api.get("tracker")
    for predicted_value in predicted_values:
        if tracker["model_type"] == "classification":
            dimensions_with_class = dimensions + [{"Name": "Class", "Value": str(predicted_value)}]
            metric = {
                "MetricName": "Prediction",
                "Dimensions": dimensions_with_class,
                "Unit": "Count",
                "Value": 1,
            }

            metric_list.append(metric)
        else:
            metric = {
                "MetricName": "Prediction",
                "Dimensions": dimensions,
                "Value": float(predicted_value),
            }
            metric_list.append(metric)
    return metric_list


def cache_classes(ctx, api, predicted_values, class_set):
    for predicted_value in predicted_values:
        if predicted_value not in class_set:
            upload_class(ctx, api["name"], predicted_value)
            class_set.add(predicted_value)


def post_request_metrics(ctx, api, response, predictions, class_set):
    api_name = api["name"]
    api_dimensions = api_metric_dimensions(ctx, api_name)
    metrics_list = []
    metrics_list += status_code_metric(api_dimensions, response.status_code)

    if predictions is not None:
        metrics_list += predictions_per_request_metric(api_dimensions, len(predictions))

        if api.get("tracker") is not None:
            try:
                predicted_values = extract_predicted_values(api, predictions)

                if api["tracker"]["model_type"] == "classification":
                    cache_classes(ctx, api, predicted_values, class_set)

                metrics_list += prediction_metrics(api_dimensions, api, predicted_values)
            except Exception as e:
                logger.warn(str(e), exc_info=True)

    try:
        ctx.publish_metrics(metrics_list)
    except Exception as e:
        logger.warn(str(e), exc_info=True)
