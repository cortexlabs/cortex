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


from cortex.lib.exceptions import UserException, CortexException
from cortex.lib.log import get_logger

logger = get_logger()


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


def prediction_metrics(dimensions, api, predictions):
    metric_list = []
    tracker = api.get("tracker")
    for prediction in predictions:
        predicted_value = prediction.get(tracker["key"])
        if predicted_value is None:
            logger.warn("key {} not found in response payload".format(tracker["key"]))
            return []

        if tracker["model_type"] == "classification":
            if type(predicted_value) == str or type(predicted_value) == int:
                dimensions_with_class = dimensions + [
                    {"Name": "Class", "Value": str(predicted_value)}
                ]
                metric = {
                    "MetricName": "Prediction",
                    "Dimensions": dimensions_with_class,
                    "Unit": "Count",
                    "Value": 1,
                }

                metric_list.append(metric)
            else:
                logger.warn(
                    "failed to track key '{}': expected type 'str' or 'int' but encountered '{}'".format(
                        tracker["key"], type(predicted_value)
                    )
                )
                return []
        else:
            if type(predicted_value) == float or type(predicted_value) == int:  # allow ints
                metric = {
                    "MetricName": "Prediction",
                    "Dimensions": dimensions,
                    "Value": float(predicted_value),
                }
                metric_list.append(metric)
            else:
                logger.warn(
                    "failed to track key '{}': expected type 'float' or 'int' but encountered '{}'".format(
                        tracker["key"], type(predicted_value)
                    )
                )
                return []
    return metric_list


def post_request_metrics(ctx, api, response, predictions):
    try:
        api_name = api["name"]

        api_dimensions = api_metric_dimensions(ctx, api_name)
        metrics_list = []
        metrics_list += status_code_metric(api_dimensions, response.status_code)

        if predictions is not None:
            metrics_list += predictions_per_request_metric(api_dimensions, len(predictions))

            if api.get("tracker") is not None:
                metrics_list += prediction_metrics(api_dimensions, api, predictions)
        ctx.publish_metrics(metrics_list)

    except CortexException as e:
        e.wrap("error")
        logger.warn(str(e), exc_info=True)
    except Exception as e:
        logger.warn(str(e), exc_info=True)
