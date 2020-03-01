# Copyright 2020 Cortex Labs, Inc.
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

import datadog

from cortex.lib.log import cx_logger
from cortex.lib.exceptions import CortexException
from cortex.lib.type.predictor import Predictor
from cortex.lib.type.tracker import Tracker


class API:
    def __init__(self, storage, cache_dir=".", **kwargs):
        self.id = kwargs["id"]
        self.key = kwargs["key"]
        self.metadata_root = kwargs["metadata_root"]
        self.name = kwargs["name"]
        self.endpoint = kwargs["endpoint"]
        self.predictor = Predictor(storage, cache_dir, **kwargs["predictor"])
        self.tracker = None
        if kwargs.get("tracker") is not None:
            self.tracker = Tracker(**kwargs["tracker"])

        self.cache_dir = cache_dir
        self.storage = storage

        host_ip = os.environ["HOST_IP"]
        datadog.initialize(statsd_host=host_ip, statsd_port="8125")
        self.statsd = datadog.statsd

    def get_cached_classes(self):
        prefix = os.path.join(self.metadata_root, self.id, "classes") + "/"
        class_paths = self.storage.search(prefix=prefix)
        class_set = set()
        for class_path in class_paths:
            encoded_class_name = class_path.split("/")[-1]
            class_set.add(base64.urlsafe_b64decode(encoded_class_name.encode()).decode())
        return class_set

    def upload_class(self, class_name):
        try:
            ascii_encoded = class_name.encode("ascii")  # cloudwatch only supports ascii
            encoded_class_name = base64.urlsafe_b64encode(ascii_encoded)
            key = os.path.join(self.metadata_root, self.id, "classes", encoded_class_name.decode())
            self.storage.put_json("", key)
        except Exception as e:
            raise ValueError("unable to store class {}".format(class_name)) from e

    def metric_dimensions(self):
        return [{"Name": "APIName", "Value": self.name}, {"Name": "APIID", "Value": self.id}]

    def post_latency_metrics(self, status_code, total_time):
        metrics = [self.status_code_metric(status_code), self.latency_metric(total_time)]
        self.post_metrics(metrics)

    def post_tracker_metrics(self, prediction_value=None):
        if prediction_value is not None:
            metrics = [self.prediction_metrics(prediction_value)]
            self.post_metrics(metrics)

    def post_metrics(self, metrics):
        try:
            if self.statsd is None:
                raise CortexException("statsd client not initialized")  # unexpected

            for metric in metrics:
                tags = ["{}:{}".format(dim["Name"], dim["Value"]) for dim in metric["Dimensions"]]
                if metric.get("Unit") == "Count":
                    self.statsd.increment(metric["MetricName"], value=metric["Value"], tags=tags)
                else:
                    self.statsd.histogram(metric["MetricName"], value=metric["Value"], tags=tags)
        except:
            cx_logger().warn("failure encountered while publishing metrics", exc_info=True)

    def status_code_metric(self, status_code):
        status_code_series = int(status_code / 100)
        status_code_dimensions = self.metric_dimensions() + [
            {"Name": "Code", "Value": "{}XX".format(status_code_series)}
        ]
        return {
            "MetricName": "StatusCode",
            "Dimensions": status_code_dimensions,
            "Value": 1,
            "Unit": "Count",
        }

    def latency_metric(self, total_time):
        return {
            "MetricName": "Latency",
            "Dimensions": self.metric_dimensions(),
            "Value": total_time * 1000,  # milliseconds
        }

    def prediction_metrics(self, prediction_value):
        if self.tracker.model_type == "classification":
            dimensions_with_class = self.metric_dimensions() + [
                {"Name": "Class", "Value": str(prediction_value)}
            ]
            return {
                "MetricName": "Prediction",
                "Dimensions": dimensions_with_class,
                "Unit": "Count",
                "Value": 1,
            }
        else:
            return {
                "MetricName": "Prediction",
                "Dimensions": self.metric_dimensions(),
                "Value": float(prediction_value),
            }
