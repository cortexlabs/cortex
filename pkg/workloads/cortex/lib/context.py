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
import imp
import inspect
import boto3

from cortex import consts
from cortex.lib import util
from cortex.lib.storage import S3, LocalStorage
from cortex.lib.exceptions import CortexException, UserException
from cortex.lib.resources import ResourceMap
from cortex.lib.log import get_logger

logger = get_logger()


class Context:
    def __init__(self, **kwargs):
        if "cache_dir" in kwargs:
            self.cache_dir = kwargs["cache_dir"]
        elif "local_path" in kwargs:
            local_path_dir = os.path.dirname(os.path.abspath(kwargs["local_path"]))
            self.cache_dir = os.path.join(local_path_dir, "cache")
        else:
            raise ValueError("cache_dir must be specified (or inferred from local_path)")
        util.mkdir_p(self.cache_dir)

        if "local_path" in kwargs:
            self.ctx = util.read_msgpack(kwargs["local_path"])
        elif "obj" in kwargs:
            self.ctx = kwargs["obj"]
        elif "raw_obj" in kwargs:
            self.ctx = kwargs["raw_obj"]
        elif "s3_path":
            local_ctx_path = os.path.join(self.cache_dir, "context.msgpack")
            bucket, key = S3.deconstruct_s3_path(kwargs["s3_path"])
            S3(bucket, client_config={}).download_file(key, local_ctx_path)
            self.ctx = util.read_msgpack(local_ctx_path)
        else:
            raise ValueError("invalid context args: " + kwargs)

        self.workload_id = kwargs.get("workload_id")

        self.id = self.ctx["id"]
        self.key = self.ctx["key"]
        self.cortex_config = self.ctx["cortex_config"]
        self.deployment_version = self.ctx["deployment_version"]
        self.root = self.ctx["root"]
        self.status_prefix = self.ctx["status_prefix"]
        self.app = self.ctx["app"]
        self.python_packages = self.ctx["python_packages"] or {}
        self.apis = self.ctx["apis"] or {}
        self.api_version = self.cortex_config["api_version"]
        self.monitoring = None

        if "local_storage_path" in kwargs:
            self.storage = LocalStorage(base_dir=kwargs["local_storage_path"])
        else:
            self.storage = S3(
                bucket=self.cortex_config["bucket"],
                region=self.cortex_config["region"],
                client_config={},
            )
            self.monitoring = boto3.client("cloudwatch", region_name=self.cortex_config["region"])

        if self.api_version != consts.CORTEX_VERSION:
            raise ValueError(
                "API version mismatch (Context: {}, Image: {})".format(
                    self.api_version, consts.CORTEX_VERSION
                )
            )

        # Internal caches
        self._metadatas = {}

        # This affects Tensorflow S3 access
        os.environ["AWS_REGION"] = self.cortex_config.get("region", "")

        # ID maps
        self.pp_id_map = ResourceMap(self.python_packages) if self.python_packages else None
        self.apis_id_map = ResourceMap(self.apis) if self.apis else None
        self.id_map = util.merge_dicts_overwrite(self.pp_id_map, self.apis_id_map)

    def download_file(self, impl_key, cache_impl_path):
        if not os.path.isfile(cache_impl_path):
            self.storage.download_file(impl_key, cache_impl_path)
        return cache_impl_path

    def download_python_file(self, impl_key, module_name):
        cache_impl_path = os.path.join(self.cache_dir, "{}.py".format(module_name))
        self.download_file(impl_key, cache_impl_path)
        return cache_impl_path

    def load_module(self, module_prefix, module_name, impl_key):
        full_module_name = "{}_{}".format(module_prefix, module_name)

        try:
            impl_path = self.download_python_file(impl_key, full_module_name)
        except CortexException as e:
            e.wrap("unable to find python file")
            raise

        try:
            impl = imp.load_source(full_module_name, impl_path)
        except Exception as e:
            raise UserException("unable to load python file") from e

        return impl, impl_path

    def get_request_handler_impl(self, api_name):
        api = self.apis[api_name]

        module_prefix = "request_handler"

        try:
            impl, impl_path = self.load_module(
                module_prefix, api["name"], api["request_handler_impl_key"]
            )
        except CortexException as e:
            e.wrap("api " + api_name, "request_handler " + api["request_handler"])
            raise

        try:
            _validate_impl(impl, REQUEST_HANDLER_IMPL_VALIDATION)
        except CortexException as e:
            e.wrap("api " + api_name, "request_handler " + api["request_handler"])
            raise
        return impl

    def get_resource_status(self, resource):
        key = self.resource_status_key(resource)
        return self.storage.get_json(key, num_retries=5)

    def upload_resource_status_start(self, *resources):
        timestamp = util.now_timestamp_rfc_3339()
        for resource in resources:
            key = self.resource_status_key(resource)
            status = {
                "resource_id": resource["id"],
                "resource_type": resource["resource_type"],
                "workload_id": resource["workload_id"],
                "app_name": self.app["name"],
                "start": timestamp,
            }
            self.storage.put_json(status, key)

    def upload_resource_status_no_op(self, *resources):
        timestamp = util.now_timestamp_rfc_3339()
        for resource in resources:
            key = self.resource_status_key(resource)
            status = {
                "resource_id": resource["id"],
                "resource_type": resource["resource_type"],
                "workload_id": resource["workload_id"],
                "app_name": self.app["name"],
                "start": timestamp,
                "end": timestamp,
                "exit_code": "succeeded",
            }
            self.storage.put_json(status, key)

    def upload_resource_status_success(self, *resources):
        self.upload_resource_status_end("succeeded", *resources)

    def upload_resource_status_failed(self, *resources):
        self.upload_resource_status_end("failed", *resources)

    def upload_resource_status_end(self, exit_code, *resources):
        timestamp = util.now_timestamp_rfc_3339()
        for resource in resources:
            status = self.get_resource_status(resource)
            if status.get("end") != None:
                continue
            status["end"] = timestamp
            status["exit_code"] = exit_code
            key = self.resource_status_key(resource)
            self.storage.put_json(status, key)

    def resource_status_key(self, resource):
        return os.path.join(self.status_prefix, resource["id"], resource["workload_id"])

    def get_metadata_url(self, resource_id):
        return os.path.join(self.ctx["metadata_root"], resource_id + ".json")

    def write_metadata(self, resource_id, metadata):
        if resource_id in self._metadatas and self._metadatas[resource_id] == metadata:
            return

        self._metadatas[resource_id] = metadata
        self.storage.put_json(metadata, self.get_metadata_url(resource_id))

    def get_metadata(self, resource_id, use_cache=True):
        if use_cache and resource_id in self._metadatas:
            return self._metadatas[resource_id]

        metadata = self.storage.get_json(self.get_metadata_url(resource_id), allow_missing=True)
        self._metadatas[resource_id] = metadata
        return metadata

    def publish_metrics(self, metrics):
        if self.monitoring is None:
            raise CortexException("monitoring client not initialized")  # unexpected

        response = self.monitoring.put_metric_data(
            MetricData=metrics, Namespace=self.cortex_config["log_group"]
        )

        if int(response["ResponseMetadata"]["HTTPStatusCode"] / 100) != 2:
            logger.warn(response)
            raise Exception("failed to publish metrics")


REQUEST_HANDLER_IMPL_VALIDATION = {
    "optional": [
        {"name": "pre_inference", "args": ["sample", "metadata"]},
        {"name": "post_inference", "args": ["prediction", "metadata"]},
    ]
}


def _validate_impl(impl, impl_req):
    for optional_func in impl_req.get("optional", []):
        _validate_optional_fn_args(impl, optional_func["name"], optional_func["args"])

    for required_func in impl_req.get("required", []):
        _validate_required_fn_args(impl, required_func["name"], required_func["args"])


def _validate_required_fn(impl, fn_name):
    _validate_required_fn_args(impl, fn_name, None)


def _validate_optional_fn_args(impl, fn_name, args):
    if hasattr(impl, fn_name):
        _validate_required_fn_args(impl, fn_name, args)


def _validate_required_fn_args(impl, fn_name, args):
    fn = getattr(impl, fn_name, None)
    if not fn:
        raise UserException("function " + fn_name, "could not find function")

    if not callable(fn):
        raise UserException("function " + fn_name, "not a function")

    argspec = inspect.getargspec(fn)
    if argspec.varargs != None or argspec.keywords != None or argspec.defaults != None:
        raise UserException(
            "function " + fn_name,
            "invalid function signature, can only accept positional arguments",
        )

    if args:
        if argspec.args != args:
            raise UserException(
                "function " + fn_name,
                "expected function arguments arguments ({}) but found ({})".format(
                    ", ".join(args), ", ".join(argspec.args)
                ),
            )
