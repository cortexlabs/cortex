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
import imp
import inspect

import dill

from cortex.lib.log import refresh_logger, cx_logger
from cortex.lib.exceptions import CortexException, UserException, UserRuntimeException
from cortex import consts

from typing import Any


class Predictor:
    def __init__(self, provider, model_dir, cache_dir, api_spec):
        self.provider = provider

        self.type = api_spec["predictor"]["type"]
        self.path = api_spec["predictor"]["path"]
        self.python_path = api_spec["predictor"].get("python_path")
        self.config = api_spec["predictor"].get("config", {})
        self.env = api_spec["predictor"].get("env")

        self.model_dir = model_dir
        self.cache_dir = cache_dir
        self.api_spec = api_spec

    def initialize_client(self, tf_serving_host=None, tf_serving_port=None):
        signature_message = None
        if self.type == "onnx":
            from cortex.lib.client.onnx import ONNXClient

            client = ONNXClient(self.models)
            if self.models[0].name == consts.SINGLE_MODEL_NAME:
                signature_message = "ONNX model signature: {}".format(
                    client.input_signatures[consts.SINGLE_MODEL_NAME]
                )
            else:
                signature_message = "ONNX model signatures: {}".format(client.input_signatures)
            cx_logger().info(signature_message)
            return client
        elif self.type == "tensorflow":
            from cortex.lib.client.tensorflow import TensorFlowClient

            for model in self.models:
                validate_model_dir(model.base_path)

            tf_serving_address = tf_serving_host + ":" + tf_serving_port
            client = TensorFlowClient(tf_serving_address, self.models)
            if self.models[0].name == consts.SINGLE_MODEL_NAME:
                signature_message = "TensorFlow model signature: {}".format(
                    client.input_signatures[consts.SINGLE_MODEL_NAME]
                )
            else:
                signature_message = "TensorFlow model signatures: {}".format(
                    client.input_signatures
                )
            cx_logger().info(signature_message)
            return client

        return None

    def initialize_impl(self, project_dir, client=None):
        class_impl = self.class_impl(project_dir)
        try:
            if self.type == "onnx":
                return class_impl(onnx_client=client, config=self.config)
            elif self.type == "tensorflow":
                return class_impl(tensorflow_client=client, config=self.config)
            else:
                return class_impl(config=self.config)
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

            _validate_impl(predictor_class, validations, self._api_spec)
        except CortexException as e:
            e.wrap("error in " + self.path)
            raise
        return predictor_class

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

    def _compute_model_basepath(self, model_path, model_name):
        base_path = os.path.join(self.model_dir, model_name)
        if self.type == "onnx":
            base_path = os.path.join(base_path, os.path.basename(model_path))
        return base_path


def _condition_specified_models(impl: Any, api_spec: dict) -> bool:
    if (
        api_spec["predictor"]["model_path"] is not None
        or api_spec["predictor"]["models"]["dir"] is not None
        or len(api_spec["predictor"]["models"]["paths"]) > 0
    ):
        return True
    return False


PYTHON_CLASS_VALIDATION = {
    "required": [
        {
            "name": "__init__",
            "required_args": ["self", "config"],
            "condition_args": {"python_client": _condition_specified_models},
        },
        {
            "name": "predict",
            "required_args": ["self"],
            "optional_args": ["payload", "query_params", "headers"],
        },
    ],
    "conditional": [
        {
            "name": "load_model",
            "required_args": ["self", "model_path"],
            "condition": _condition_specified_models,
        }
    ],
}

TENSORFLOW_CLASS_VALIDATION = {
    "required": [
        {"name": "__init__", "required_args": ["self", "tensorflow_client", "config"]},
        {
            "name": "predict",
            "required_args": ["self"],
            "optional_args": ["payload", "query_params", "headers"],
        },
    ]
}

ONNX_CLASS_VALIDATION = {
    "required": [
        {"name": "__init__", "required_args": ["self", "onnx_client", "config"]},
        {
            "name": "predict",
            "required_args": ["self"],
            "optional_args": ["payload", "query_params", "headers"],
        },
    ]
}


def _validate_impl(impl, impl_req, api_spec):
    for optional_func_signature in impl_req.get("optional", []):
        _validate_optional_fn_args(impl, optional_func_signature)

    for required_func_signature in impl_req.get("required", []):
        _validate_required_fn_args(impl, required_func_signature)

    for required_func_signature in impl_req.get("conditional", []):
        if required_func_signature["condition"](impl, api_spec):
            _validate_required_fn_args(impl, required_func_signature)


def _validate_optional_fn_args(impl, func_signature):
    if getattr(impl, func_signature["name"], None):
        _validate_required_fn_args(impl, func_signature)


def _validate_required_fn_args(impl, func_signature):
    fn = getattr(impl, func_signature["name"], None)
    if not fn:
        raise UserException(f'required function "{func_signature["name"]}" is not defined')

    if not callable(fn):
        raise UserException(f'"{func_signature["name"]}" is defined, but is not a function')

    required_args = func_signature.get("required_args", [])
    optional_args = func_signature.get("optional_args", [])

    if func_signature.get("condition_args"):
        for arg_name in func_signature["condition_args"]:
            if func_signature["condition_args"][arg_name](impl, api_spec):
                required_args.append(arg_name)

    argspec = inspect.getfullargspec(fn)
    fn_str = f'{func_signature["name"]}({", ".join(argspec.args)})'

    for arg_name in required_args:
        if arg_name not in argspec.args:
            raise UserException(
                f'invalid signature for function "{fn_str}": "{arg_name}" is a required argument, but was not provided'
            )

        if arg_name == "self":
            if argspec.args[0] != "self":
                raise UserException(
                    f'invalid signature for function "{fn_str}": "self" must be the first argument'
                )

    seen_args = []
    for arg_name in argspec.args:
        if arg_name not in required_args and arg_name not in optional_args:
            raise UserException(
                f'invalid signature for function "{fn_str}": "{arg_name}" is not a supported argument'
            )

        if arg_name in seen_args:
            raise UserException(
                f'invalid signature for function "{fn_str}": "{arg_name}" is duplicated'
            )

        seen_args.append(arg_name)


def uses_neuron_savedmodel():
    return os.getenv("CORTEX_ACTIVE_NEURON") != None


def get_expected_dir_structure():
    if uses_neuron_savedmodel():
        return neuron_tf_expected_dir_structure
    return tf_expected_dir_structure


tf_expected_dir_structure = """tensorflow model directories must have the following structure:
  1523423423/ (version prefix, usually a timestamp)
  ├── saved_model.pb
  └── variables/
      ├── variables.index
      ├── variables.data-00000-of-00003
      ├── variables.data-00001-of-00003
      └── variables.data-00002-of-...`"""

neuron_tf_expected_dir_structure = """neuron tensorflow model directories must have the following structure:
  1523423423/ (version prefix, usually a timestamp)
  └── saved_model.pb`"""


def validate_model_dir(model_dir):
    version = None
    for file_name in os.listdir(model_dir):
        if file_name.isdigit():
            version = file_name
            break

    if version is None:
        cx_logger().error(get_expected_dir_structure())
        raise UserException("no top-level version folder found")

    if not os.path.isdir(os.path.join(model_dir, version)):
        cx_logger().error(get_expected_dir_structure())
        raise UserException("no top-level version folder found")

    if not os.path.isfile(os.path.join(model_dir, version, "saved_model.pb")):
        cx_logger().error(get_expected_dir_structure())
        raise UserException('expected a "saved_model.pb" file')

    if not uses_neuron_savedmodel():
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
