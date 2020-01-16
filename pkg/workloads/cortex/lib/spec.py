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
import datadog
import dill

from cortex import consts
from cortex.lib import util, api_utils
from cortex.lib.log import refresh_logger
from cortex.lib.storage import S3, LocalStorage
from cortex.lib.exceptions import CortexException, UserException
from cortex.lib.resources import ResourceMap


"""
local_ctx_path = os.path.join(self.cache_dir, "context.msgpack")
bucket, key = S3.deconstruct_s3_path(kwargs["s3_path"])
S3(bucket, client_config={}).download_file(key, local_ctx_path)
self.ctx = util.read_msgpack(local_ctx_path)

host_ip = os.environ["HOST_IP"]
datadog.initialize(statsd_host=host_ip, statsd_port="8125")
self.statsd = datadog.statsd

if self.api_version != consts.CORTEX_VERSION:
    raise ValueError(
        "api version mismatch (context: {}, image: {})".format(
            self.api_version, consts.CORTEX_VERSION
        )
    )

os.environ["AWS_REGION"] = self.cluster_config.get("region", "")
"""


class API:
    def __init__(self, storage, cache_dir=".", **kwargs):
        self.id = kwargs["id"]
        self.key = kwargs["key"]
        self.metadata_root = kwargs["metadata_root"]
        self.name = kwargs["name"]
        self.endpoint = kwargs["endpoint"]
        self.predictor = Predictor(storage, cache_dir, **kwargs["predictor"])
        self.tracker = None

        if kwargs.get("tracker") is None:
            self.tracker = Tracker(**kwargs["tracker"])

        self.cache_dir = cache_dir
        self.statsd = api_utils.get_statsd_client()
        self.storage = storage

        self._spec = kwargs


class Tracker:
    def __init__(self, **kwargs):
        self.key = kwargs.get("key")
        self.model_type = kwargs["model_type"]


class Predictor:
    def __init__(self, storage, cache_dir, **kwargs):
        print(kwargs)
        print(kwargs["path"])
        self.type = kwargs["type"]
        self.path = kwargs["path"]
        self.model = kwargs.get("model")
        self.python_path = kwargs.get("python_path")
        self.config = kwargs.get("config", {})
        self.env = kwargs.get("env")
        self.signature_key = kwargs.get("signature_key")

        self.cache_dir = cache_dir
        self.storage = storage

    def _download_file(self, impl_key, cache_impl_path):
        if not os.path.isfile(cache_impl_path):
            self.storage.download_file(impl_key, cache_impl_path)
        return cache_impl_path

    def _download_python_file(self, impl_key, module_name):
        cache_impl_path = os.path.join(self.cache_dir, "{}.py".format(module_name))
        self._download_file(impl_key, cache_impl_path)
        return cache_impl_path

    def _load_module(self, module_name, impl_path):
        if impl_path.endswith(".pickle"):
            try:
                impl = imp.new_module(module_name)

                with open(impl_path, "rb") as pickle_file:
                    pickled_dict = dill.load(pickle_file)
                    for key in pickled_dict:
                        setattr(impl, key, pickled_dict[key])
            except Exception as e:
                raise UserException("unable to load pickle", str(e)) from e
        else:
            try:
                impl = imp.load_source(module_name, impl_path)
            except Exception as e:
                raise UserException(str(e)) from e

        return impl

    def class_impl(self, project_dir):
        if self.type == "tensorflow":
            target_class_name = "TensorFlowPredictor"
            validations = TENSORFLOW_CLASS_VALIDATION
        elif self.type == "onnx":
            target_class_name = "ONNXPredictor"
            validations = ONNX_CLASS_VALIDATION
        elif self.type == "python":
            target_class_name = "PythonPredictor"
            validations = PYTHON_CLASS_VALIDATION

        try:
            impl = self._load_module("cortex_predictor", os.path.join(project_dir, self.path))
        except CortexException as e:
            e.wrap("error in " + self.path)
            raise
        finally:
            refresh_logger()

        try:
            classes = inspect.getmembers(impl, inspect.isclass)
            predictor_class = None
            for class_df in classes:
                if class_df[0] == target_class_name:
                    if predictor_class is not None:
                        raise UserException(
                            "multiple definitions for {} class found; please check your imports and class definitions and ensure that there is only one Predictor class definition".format(
                                target_class_name
                            )
                        )
                    predictor_class = class_df[1]
            if predictor_class is None:
                raise UserException("{} class is not defined".format(target_class_name))

            _validate_impl(predictor_class, validations)
        except CortexException as e:
            e.wrap("error in " + self.path)
            raise
        return predictor_class


PYTHON_CLASS_VALIDATION = {
    "required": [
        {"name": "__init__", "args": ["self", "config"]},
        {"name": "predict", "args": ["self", "payload"]},
    ]
}

TENSORFLOW_CLASS_VALIDATION = {
    "required": [
        {"name": "__init__", "args": ["self", "tensorflow_client", "config"]},
        {"name": "predict", "args": ["self", "payload"]},
    ]
}

ONNX_CLASS_VALIDATION = {
    "required": [
        {"name": "__init__", "args": ["self", "onnx_client", "config"]},
        {"name": "predict", "args": ["self", "payload"]},
    ]
}


def _validate_impl(impl, impl_req):
    for optional_func in impl_req.get("optional", []):
        _validate_optional_fn_args(impl, optional_func["name"], optional_func["args"])

    for required_func in impl_req.get("required", []):
        _validate_required_fn_args(impl, required_func["name"], required_func["args"])


def _validate_optional_fn_args(impl, fn_name, args):
    if fn_name in vars(impl):
        _validate_required_fn_args(impl, fn_name, args)


def _validate_required_fn_args(impl, fn_name, args):
    fn = getattr(impl, fn_name, None)
    if not fn:
        raise UserException('required function "{}" is not defined'.format(fn_name))

    if not callable(fn):
        raise UserException('"{}" is defined, but is not a function'.format(fn_name))

    argspec = inspect.getargspec(fn)

    if argspec.args != args:
        raise UserException(
            'invalid signature for function "{}": expected arguments ({}) but found ({})'.format(
                fn_name, ", ".join(args), ", ".join(argspec.args)
            )
        )
