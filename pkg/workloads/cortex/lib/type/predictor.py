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

import dill

from cortex.lib.log import refresh_logger, cx_logger
from cortex.lib.exceptions import CortexException, UserException, UserRuntimeException


class Predictor:
    def __init__(self, storage, cache_dir, **kwargs):
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

    def initialize_client(self, args):
        if self.type == "onnx":
            from cortex.lib.client.onnx import ONNXClient

            _, prefix = self.storage.deconstruct_s3_path(self.model)
            model_path = os.path.join(args.model_dir, os.path.basename(prefix))
            client = ONNXClient(model_path)
            cx_logger().info("ONNX model signature: {}".format(client.input_signature))
            return client
        elif self.type == "tensorflow":
            from cortex.lib.client.tensorflow import TensorFlowClient

            validate_model_dir(args.model_dir)
            client = TensorFlowClient("localhost:" + str(args.tf_serve_port), self.signature_key)
            cx_logger().info("TensorFlow model signature: {}".format(client.input_signature))
            return client

        return None

    def initialize_impl(self, project_dir, client=None):
        class_impl = self.class_impl(project_dir)
        try:
            if self.type == "python":
                return class_impl(self.config)
            else:
                return class_impl(client, self.config)
        except Exception as e:
            raise UserRuntimeException(self.path, "__init__", str(e)) from e
        finally:
            refresh_logger()

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


tf_expected_dir_structure = """tensorflow model directories must have the following structure:
  1523423423/ (version prefix, usually a timestamp)
  ├── saved_model.pb
  └── variables/
      ├── variables.index
      ├── variables.data-00000-of-00003
      ├── variables.data-00001-of-00003
      └── variables.data-00002-of-...`"""


def validate_model_dir(model_dir):
    version = None
    for file_name in os.listdir(model_dir):
        if file_name.isdigit():
            version = file_name
            break

    if version is None:
        cx_logger().error(tf_expected_dir_structure)
        raise UserException("no top-level version folder found")

    if not os.path.isdir(os.path.join(model_dir, version)):
        cx_logger().error(tf_expected_dir_structure)
        raise UserException("no top-level version folder found")

    if not os.path.isfile(os.path.join(model_dir, version, "saved_model.pb")):
        cx_logger().error(tf_expected_dir_structure)
        raise UserException('expected a "saved_model.pb" file')

    if not os.path.isdir(os.path.join(model_dir, version, "variables")):
        cx_logger().error(tf_expected_dir_structure)
        raise UserException('expected a "variables" directory')

    if not os.path.isfile(os.path.join(model_dir, version, "variables", "variables.index")):
        cx_logger().error(tf_expected_dir_structure)
        raise UserException('expected a "variables/variables.index" file')

    for file_name in os.listdir(os.path.join(model_dir, version, "variables")):
        if file_name.startswith("variables.data-00000-of"):
            return

    cx_logger().error(tf_expected_dir_structure)
    raise UserException(
        'expected at least one variables data file, starting with "variables.data-00000-of-"'
    )
