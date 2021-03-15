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

import imp
import inspect
import json
import os
from copy import deepcopy
from http import HTTPStatus
from typing import Any, Dict, Union

import datadog
import dill

from cortex_internal.lib.api.validations import validate_class_impl
from cortex_internal.lib.exceptions import CortexException, UserException, UserRuntimeException
from cortex_internal.lib.metrics import MetricsClient
from cortex_internal.lib.storage import S3

ASYNC_PYTHON_PREDICTOR_VALIDATION = {
    "required": [
        {
            "name": "__init__",
            "required_args": ["self", "config"],
            "optional_args": ["metrics_client"],
        },
        {
            "name": "predict",
            "required_args": ["self"],
            "optional_args": ["payload", "request_id"],
        },
    ],
}


class AsyncAPI:
    def __init__(
        self,
        api_spec: Dict[str, Any],
        storage: S3,
        storage_path: str,
        statsd_host: str,
        statsd_port: int,
    ):
        self.api_spec = api_spec
        self.storage = storage
        self.storage_path = storage_path
        self.path = api_spec["predictor"]["path"]
        self.config = api_spec["predictor"].get("config", {})

        datadog.initialize(statsd_host=statsd_host, statsd_port=statsd_port)
        self.__statsd = datadog.statsd

    @property
    def statsd(self):
        return self.__statsd

    def update_status(self, request_id: str, status: str):
        self.storage.put_str(status, f"{self.storage_path}/{request_id}/status")

    def upload_result(self, request_id: str, result: Dict[str, Any]):
        if not isinstance(result, dict):
            raise UserRuntimeException(
                f"user response must be json serializable dictionary, got {type(result)} instead"
            )

        try:
            result_json = json.dumps(result)
        except Exception:
            raise UserRuntimeException("user response is not json serializable")

        self.storage.put_str(result_json, f"{self.storage_path}/{request_id}/result.json")

    def get_payload(self, request_id: str) -> Union[Dict, str, bytes]:
        key = f"{self.storage_path}/{request_id}/payload"
        obj = self.storage.get_object(key)
        status_code = obj["ResponseMetadata"]["HTTPStatusCode"]
        if status_code != HTTPStatus.OK:
            raise CortexException(
                f"failed to retrieve async payload (request_id: {request_id}, status_code: {status_code})"
            )

        content_type: str = obj["ResponseMetadata"]["HTTPHeaders"]["content-type"]
        payload_bytes: bytes = obj["Body"].read()

        # decode payload
        if content_type.startswith("application/json"):
            try:
                return json.loads(payload_bytes)
            except Exception as err:
                raise UserRuntimeException(
                    f"the uploaded payload, with content-type {content_type}, could not be decoded to JSON"
                ) from err
        elif content_type.startswith("text/plain"):
            try:
                return payload_bytes.decode("utf-8")
            except Exception as err:
                raise UserRuntimeException(
                    f"the uploaded payload, with content-type {content_type}, could not be decoded to a utf-8 string"
                ) from err
        else:
            return payload_bytes

    def delete_payload(self, request_id: str):
        key = f"{self.storage_path}/{request_id}/payload"
        self.storage.delete(key)

    def initialize_impl(self, project_dir: str, metrics_client: MetricsClient):
        predictor_impl = self._get_impl(project_dir)
        constructor_args = inspect.getfullargspec(predictor_impl.__init__).args
        config = deepcopy(self.config)

        args = {}
        if "config" in constructor_args:
            args["config"] = config
        if "metrics_client" in constructor_args:
            args["metrics_client"] = metrics_client

        try:
            predictor = predictor_impl(**args)
        except Exception as e:
            raise UserRuntimeException(self.path, "__init__", str(e)) from e

        return predictor

    def _get_impl(self, project_dir: str):
        try:
            impl = self._read_impl(
                "cortex_async_predictor", os.path.join(project_dir, self.path), "PythonPredictor"
            )
        except CortexException as e:
            e.wrap("error in " + self.path)
            raise

        try:
            self._validate_impl(impl)
        except CortexException as e:
            e.wrap("error in " + self.path)
            raise

        return impl

    @staticmethod
    def _read_impl(module_name: str, impl_path: str, target_class_name):
        if impl_path.endswith(".pickle"):
            try:
                with open(impl_path, "rb") as pickle_file:
                    return dill.load(pickle_file)
            except Exception as e:
                raise UserException("unable to load pickle", str(e)) from e

        try:
            impl = imp.load_source(module_name, impl_path)
        except Exception as e:
            raise UserException(str(e)) from e

        classes = inspect.getmembers(impl, inspect.isclass)

        predictor_class = None
        for class_df in classes:
            if class_df[0] == target_class_name:
                if predictor_class is not None:
                    raise UserException(
                        f"multiple definitions for {target_class_name} class found; please check "
                        f"your imports and class definitions and ensure that there is only one "
                        f"predictor class definition"
                    )
                predictor_class = class_df[1]

        if predictor_class is None:
            raise UserException(f"{target_class_name} class is not defined")

        return predictor_class

    @staticmethod
    def _validate_impl(impl):
        return validate_class_impl(impl, ASYNC_PYTHON_PREDICTOR_VALIDATION)
