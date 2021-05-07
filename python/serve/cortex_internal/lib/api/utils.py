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

import datetime
import glob
import itertools
import json
import os
import shutil
import threading as td
import time
import traceback
from collections import defaultdict
from http import HTTPStatus
from typing import Any, Callable, Dict, List, Optional, Tuple

from datadog.dogstatsd.base import DogStatsd
from starlette.responses import Response

from cortex_internal.lib import util
from cortex_internal.lib.exceptions import CortexException, UserRuntimeException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.model import validate_model_paths
from cortex_internal.lib.storage import S3
from cortex_internal.lib.type import HandlerType

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


def _read_json(json_path: str):
    with open(json_path) as json_file:
        return json.load(json_file)


def get_spec(
    spec_path: str,
    cache_dir: str,
    region: str,
    spec_name: str = "api_spec.json",
) -> Tuple[S3, dict]:
    """
    Args:
        spec_path: Path to API spec (i.e. "s3://cortex-dev-0/apis/iris-classifier/api/69b93378fa5c0218-jy1fjtyihu-9fcc10739e7fc8050cefa8ca27ece1ee/master-spec.json").
        cache_dir: Local directory where the API spec gets saved to.
        region: Region of the bucket.
        spec_name: File name of the spec as it is saved on disk.
    """

    bucket, key = S3.deconstruct_s3_path(spec_path)
    storage = S3(bucket=bucket, region=region)

    local_spec_path = os.path.join(cache_dir, spec_name)
    if not os.path.isfile(local_spec_path):
        storage.download_file(key, local_spec_path)

    return storage, _read_json(local_spec_path)


def model_downloader(
    handler_type: HandlerType,
    bucket_name: str,
    model_name: str,
    model_version: str,
    model_path: str,
    temp_dir: str,
    model_dir: str,
) -> Optional[datetime.datetime]:
    """
    Downloads model to disk. Validates the s3 model path and the downloaded model.

    Args:
        handler_type: The handler type as implemented by the API.
        bucket_name: Name of the bucket where the model is stored.
        model_name: Name of the model. Is part of the model's local path.
        model_version: Version of the model. Is part of the model's local path.
        model_path: Model prefix of the versioned model.
        temp_dir: Where to temporarily store the model for validation.
        model_dir: The top directory of where all models are stored locally.

    Returns:
        The model's timestamp. None if the model didn't pass the validation, if it doesn't exist or if there are not enough permissions.
    """

    logger.info(
        f"downloading from bucket {bucket_name}/{model_path}, model {model_name} of version {model_version}, temporarily to {temp_dir} and then finally to {model_dir}"
    )

    client = S3(bucket_name)

    # validate upstream S3 model
    sub_paths, ts = client.search(model_path)
    try:
        validate_model_paths(sub_paths, handler_type, model_path)
    except CortexException:
        logger.info(f"failed validating model {model_name} of version {model_version}")
        return None

    # download model to temp dir
    temp_dest = os.path.join(temp_dir, model_name, model_version)
    try:
        client.download_dir_contents(model_path, temp_dest)
    except CortexException:
        logger.info(
            f"failed downloading model {model_name} of version {model_version} to temp dir {temp_dest}"
        )
        shutil.rmtree(temp_dest)
        return None

    # validate model
    model_contents = glob.glob(os.path.join(temp_dest, "**"), recursive=True)
    model_contents = util.remove_non_empty_directory_paths(model_contents)
    try:
        validate_model_paths(model_contents, handler_type, temp_dest)
    except CortexException:
        logger.info(
            f"failed validating model {model_name} of version {model_version} from temp dir"
        )
        shutil.rmtree(temp_dest)
        return None

    # move model to dest dir
    model_top_dir = os.path.join(model_dir, model_name)
    ondisk_model_version = os.path.join(model_top_dir, model_version)
    logger.info(
        f"moving model {model_name} of version {model_version} to final dir {ondisk_model_version}"
    )
    if os.path.isdir(ondisk_model_version):
        shutil.rmtree(ondisk_model_version)
    shutil.move(temp_dest, ondisk_model_version)

    return max(ts)


class CortexMetrics:
    def __init__(
        self,
        statsd_client: DogStatsd,
        api_spec: Dict[str, Any],
    ):
        self._metric_value_id = api_spec["id"]
        self._metric_value_handler_id = api_spec["handler_id"]
        self._metric_value_deployment_id = api_spec["deployment_id"]
        self._metric_value_name = api_spec["name"]
        self.__statsd = statsd_client

    def metric_dimensions_with_id(self):
        return [
            {"Name": "api_name", "Value": self._metric_value_name},
            {"Name": "api_id", "Value": self._metric_value_id},
            {"Name": "handler_id", "Value": self._metric_value_handler_id},
            {"Name": "deployment_id", "Value": self._metric_value_deployment_id},
        ]

    def metric_dimensions(self):
        return [{"Name": "api_name", "Value": self._metric_value_name}]

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
            if self.__statsd is None:
                raise CortexException("statsd client not initialized")  # unexpected

            for metric in metrics:
                tags = ["{}:{}".format(dim["Name"], dim["Value"]) for dim in metric["Dimensions"]]
                if metric.get("Unit") == "Count":
                    self.__statsd.increment(metric["MetricName"], value=metric["Value"], tags=tags)
                else:
                    self.__statsd.histogram(metric["MetricName"], value=metric["Value"], tags=tags)
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


class DynamicBatcher:
    def __init__(
        self,
        handler_impl: Callable,
        method_name: str,
        max_batch_size: int,
        batch_interval: int,
        test_mode: bool = False,
    ):
        self.method_name = method_name
        self.handler_impl = handler_impl

        self.batch_max_size = max_batch_size
        self.batch_interval = batch_interval  # measured in seconds
        self.test_mode = test_mode  # only for unit testing
        self._test_batch_lengths = []  # only when unit testing

        self.barrier = td.Barrier(self.batch_max_size + 1)

        self.samples = {}
        self.results = {}
        td.Thread(target=self._batch_engine, daemon=True).start()

        self.sample_id_generator = itertools.count()

    def _batch_engine(self):
        while True:
            if len(self.results) > 0:
                time.sleep(0.001)
                continue

            try:
                self.barrier.wait(self.batch_interval)
            except td.BrokenBarrierError:
                pass

            self.results = {}
            sample_ids = self._get_sample_ids(self.batch_max_size)
            try:
                if self.samples:
                    batch = self._make_batch(sample_ids)

                    results = getattr(self.handler_impl, self.method_name)(**batch)
                    if not isinstance(results, list):
                        raise UserRuntimeException(
                            f"please return a list when using server side batching, got {type(results)}"
                        )

                    if self.test_mode:
                        self._test_batch_lengths.append(len(results))

                    self.results = dict(zip(sample_ids, results))
            except Exception as e:
                self.results = {sample_id: e for sample_id in sample_ids}
                logger.error(traceback.format_exc())
            finally:
                for sample_id in sample_ids:
                    del self.samples[sample_id]
                self.barrier.reset()

    def _get_sample_ids(self, max_number: int) -> List[int]:
        if len(self.samples) <= max_number:
            return list(self.samples.keys())
        return sorted(self.samples)[:max_number]

    def _make_batch(self, sample_ids: List[int]) -> Dict[str, List[Any]]:
        batched_samples = defaultdict(list)
        for sample_id in sample_ids:
            for key, sample in self.samples[sample_id].items():
                batched_samples[key].append(sample)

        return dict(batched_samples)

    def _enqueue_request(self, sample_id: int, **kwargs):
        """
        Enqueue sample for batch processing. This is a blocking method.
        """

        self.samples[sample_id] = kwargs
        try:
            self.barrier.wait()
        except td.BrokenBarrierError:
            pass

    def process(self, **kwargs):
        """
        Queues a request to be batched with other incoming request, waits for the response
        and returns the processed result. This is a blocking method.
        """
        sample_id = next(self.sample_id_generator)
        self._enqueue_request(sample_id, **kwargs)
        result = self._get_result(sample_id)
        return result

    def _get_result(self, sample_id: int) -> Any:
        """
        Return the processed result. This is a blocking method.
        """
        while sample_id not in self.results:
            time.sleep(0.001)

        result = self.results[sample_id]
        del self.results[sample_id]

        if isinstance(result, Exception):
            return Response(
                content=str(result),
                status_code=HTTPStatus.INTERNAL_SERVER_ERROR,
                media_type="text/plain",
            )

        return result
