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

import base64
import json
import os
import threading
from pathlib import Path
from typing import Any, Dict, Optional, Tuple, Union

import datadog
from cortex.lib.api import Monitoring, Predictor
from cortex.lib.exceptions import CortexException
from cortex.lib.storage import LocalStorage, S3, GCS
from cortex.lib.log import cx_logger as logger


class API:
    def __init__(
        self,
        provider: str,
        storage: Union[LocalStorage, S3],
        api_spec: Dict[str, Any],
        model_dir: str,
        cache_dir: str = ".",
    ):
        self.provider = provider
        self.storage = storage
        self.api_spec = api_spec
        self.cache_dir = cache_dir

        self.id = api_spec["id"]
        self.predictor_id = api_spec["predictor_id"]
        self.deployment_id = api_spec["deployment_id"]

        self.key = api_spec["key"]
        self.metadata_root = api_spec["metadata_root"]
        self.name = api_spec["name"]
        self.predictor = Predictor(provider, api_spec, model_dir)

        self.monitoring = None
        if self.api_spec.get("monitoring") is not None:
            self.monitoring = Monitoring(**self.api_spec["monitoring"])

        if provider != "local":
            host_ip = os.environ["HOST_IP"]
            datadog.initialize(statsd_host=host_ip, statsd_port="8125")
            self.statsd = datadog.statsd

        if provider == "local":
            self.metrics_file_lock = threading.Lock()

    @property
    def server_side_batching_enabled(self):
        return self.api_spec["predictor"].get("server_side_batching") is not None

    def get_cached_classes(self):
        prefix = os.path.join(self.metadata_root, "classes") + "/"
        class_paths, _ = self.storage.search(prefix=prefix)
        class_set = set()
        for class_path in class_paths:
            encoded_class_name = class_path.split("/")[-1]
            class_set.add(base64.urlsafe_b64decode(encoded_class_name.encode()).decode())
        return class_set

    def upload_class(self, class_name: str):
        try:
            ascii_encoded = class_name.encode("ascii")  # cloudwatch only supports ascii
            encoded_class_name = base64.urlsafe_b64encode(ascii_encoded)
            key = os.path.join(self.metadata_root, "classes", encoded_class_name.decode())
            self.storage.put_json("", key)
        except Exception as e:
            raise ValueError("unable to store class {}".format(class_name)) from e

    def metric_dimensions_with_id(self):
        return [
            {"Name": "APIName", "Value": self.name},
            {"Name": "PredictorID", "Value": self.predictor_id},
            {"Name": "DeploymentID", "Value": self.deployment_id},
        ]

    def metric_dimensions(self):
        return [{"Name": "APIName", "Value": self.name}]

    def post_request_metrics(self, status_code, total_time):
        total_time_ms = total_time * 1000
        if self.provider == "local":
            self.store_metrics_locally(status_code, total_time_ms)
        else:
            metrics = [
                self.status_code_metric(self.metric_dimensions(), status_code),
                self.status_code_metric(self.metric_dimensions_with_id(), status_code),
                self.latency_metric(self.metric_dimensions(), total_time_ms),
                self.latency_metric(self.metric_dimensions_with_id(), total_time_ms),
            ]
            self.post_metrics(metrics)

    def post_monitoring_metrics(self, prediction_value=None):
        if prediction_value is not None:
            metrics = [
                self.prediction_metrics(self.metric_dimensions(), prediction_value),
                self.prediction_metrics(self.metric_dimensions_with_id(), prediction_value),
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
            logger().warn("failure encountered while publishing metrics", exc_info=True)

    def store_metrics_locally(self, status_code, total_time):
        status_code_series = int(status_code / 100)
        status_code_file_name = f"/mnt/workspace/{os.getpid()}.{status_code_series}XX"
        request_time_file = f"/mnt/workspace/{os.getpid()}.request_time"

        self.metrics_file_lock.acquire()
        try:
            self.increment_counter_file(status_code_file_name, 1)
            self.increment_counter_file(request_time_file, total_time)
        finally:
            self.metrics_file_lock.release()

    def increment_counter_file(self, file_name, value):
        previous_val = 0
        if Path(file_name).is_file():
            with open(file_name, "r") as f:
                previous_val = json.load(f)  # values are either of type int or float

        with open(file_name, "w") as f:
            json.dump(previous_val + value, f)

    def status_code_metric(self, dimensions, status_code):
        status_code_series = int(status_code / 100)
        status_code_dimensions = dimensions + [
            {"Name": "Code", "Value": "{}XX".format(status_code_series)}
        ]
        return {
            "MetricName": "StatusCode",
            "Dimensions": status_code_dimensions,
            "Value": 1,
            "Unit": "Count",
        }

    def latency_metric(self, dimensions, total_time):
        return {
            "MetricName": "Latency",
            "Dimensions": dimensions,
            "Value": total_time,  # milliseconds
        }

    def prediction_metrics(self, dimensions, prediction_value):
        if self.monitoring.model_type == "classification":
            dimensions_with_class = dimensions + [{"Name": "Class", "Value": str(prediction_value)}]
            return {
                "MetricName": "Prediction",
                "Dimensions": dimensions_with_class,
                "Unit": "Count",
                "Value": 1,
            }
        else:
            return {
                "MetricName": "Prediction",
                "Dimensions": dimensions,
                "Value": float(prediction_value),
            }


def get_api(
    provider: str,
    spec_path: str,
    model_dir: str,
    cache_dir: str,
    region: Optional[str] = None,
) -> API:
    storage, raw_api_spec = get_spec(provider, spec_path, cache_dir, region)

    api = API(
        provider=provider,
        storage=storage,
        api_spec=raw_api_spec,
        model_dir=model_dir,
        cache_dir=cache_dir,
    )

    return api


def get_spec(
    provider: str,
    spec_path: str,
    cache_dir: str,
    region: Optional[str] = None,
) -> Tuple[Union[LocalStorage, S3, GCS], dict]:
    """
    Args:
        provider: "local", "aws" or "gcp".
        spec_path: Path to API spec (i.e. "s3://cortex-dev-0/apis/iris-classifier/api/69b93378fa5c0218-jy1fjtyihu-9fcc10739e7fc8050cefa8ca27ece1ee/master-spec.json").
        cache_dir: Local directory where the API spec gets saved to.
        region: Region of the bucket. Only required for "S3" provider.
    """

    if provider == "local":
        storage = LocalStorage(cache_dir)
    elif provider == "aws":
        bucket, key = S3.deconstruct_s3_path(spec_path)
        storage = S3(bucket=bucket, region=region)
    elif provider == "gcp":
        bucket, key = GCS.deconstruct_gcs_path(spec_path)
        storage = GCS(bucket=bucket)
    else:
        raise ValueError('invalid "provider" argument')

    if provider == "local":
        return storage, read_json(spec_path)

    local_spec_path = os.path.join(cache_dir, "api_spec.json")
    if not os.path.isfile(local_spec_path):
        storage.download_file(key, local_spec_path)

    return storage, read_json(local_spec_path)


def read_json(json_path: str):
    with open(json_path) as json_file:
        return json.load(json_file)
