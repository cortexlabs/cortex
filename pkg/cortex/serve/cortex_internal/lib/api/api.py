# Copyright 2021 Cortex Labs, Inc.
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

import json
import os
from typing import Any, Dict, Tuple

import datadog

from cortex_internal.lib.api import Predictor
from cortex_internal.lib.exceptions import CortexException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.storage import S3

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


class API:
    def __init__(
        self,
        api_spec: Dict[str, Any],
        model_dir: str,
        cache_dir: str = ".",
    ):
        self.api_spec = api_spec
        self.cache_dir = cache_dir

        self.id = api_spec["id"]
        self.predictor_id = api_spec["predictor_id"]
        self.deployment_id = api_spec["deployment_id"]

        self.name = api_spec["name"]
        self.predictor = Predictor(api_spec, model_dir)

        host_ip = os.environ["HOST_IP"]
        datadog.initialize(statsd_host=host_ip, statsd_port=9125)
        self.__statsd = datadog.statsd

    @property
    def statsd(self):
        return self.__statsd

    @property
    def python_server_side_batching_enabled(self):
        return (
            self.api_spec["predictor"].get("server_side_batching") is not None
            and self.api_spec["predictor"]["type"] == "python"
        )

    def metric_dimensions_with_id(self):
        return [
            {"Name": "api_name", "Value": self.name},
            {"Name": "api_id", "Value": self.id},
            {"Name": "predictor_id", "Value": self.predictor_id},
            {"Name": "deployment_id", "Value": self.deployment_id},
        ]

    def metric_dimensions(self):
        return [{"Name": "api_name", "Value": self.name}]

    def post_request_metrics(self, status_code, total_time):
        total_time_ms = total_time * 1000
        metrics = [
            self.status_code_metric(self.metric_dimensions(), status_code),
            self.status_code_metric(self.metric_dimensions_with_id(), status_code),
            self.latency_metric(self.metric_dimensions(), total_time_ms),
            self.latency_metric(self.metric_dimensions_with_id(), total_time_ms),
        ]
        self.post_metrics(metrics)

    def post_status_code_request_metrics(self, status_code):
        metrics = [
            self.status_code_metric(self.metric_dimensions(), status_code),
            self.status_code_metric(self.metric_dimensions_with_id(), status_code),
        ]
        self.post_metrics(metrics)

    def post_latency_request_metrics(self, total_time):
        total_time_ms = total_time * 1000
        metrics = [
            self.latency_metric(self.metric_dimensions(), total_time_ms),
            self.latency_metric(self.metric_dimensions_with_id(), total_time_ms),
        ]
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
            logger.warn("failure encountered while publishing metrics", exc_info=True)

    def status_code_metric(self, dimensions, status_code):
        status_code_series = int(status_code / 100)
        status_code_dimensions = dimensions + [
            {"Name": "response_code", "Value": "{}XX".format(status_code_series)}
        ]
        return {
            "MetricName": "cortex_status_code",
            "Dimensions": status_code_dimensions,
            "Value": 1,
            "Unit": "Count",
        }

    def latency_metric(self, dimensions, total_time):
        return {
            "MetricName": "cortex_latency",
            "Dimensions": dimensions,
            "Value": total_time,  # milliseconds
        }


def get_api(
    spec_path: str,
    model_dir: str,
    cache_dir: str,
) -> API:
    with open(spec_path) as json_file:
        raw_api_spec = json.load(json_file)

    api = API(
        api_spec=raw_api_spec,
        model_dir=model_dir,
        cache_dir=cache_dir,
    )

    return api
