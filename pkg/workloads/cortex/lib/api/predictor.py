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
from typing import Any, Optional, Union

# types
from cortex.lib.type import (
    predictor_type_from_api_spec,
    PythonPredictorType,
    TensorFlowPredictorType,
    TensorFlowNeuronPredictorType,
    ONNXPredictorType,
)

# clients
from cortex.lib.client.python import PythonClient
from cortex.lib.client.tensorflow import TensorFlowClient
from cortex.lib.client.onnx import ONNXClient


# crons
from cortex.lib.model import (
    FileBasedModelsTreeUpdater,  # only when num workers > 1
    TFSModelLoader,
    ModelsGC,
    ModelTreeUpdater,
    ModelPreloader,
)

# structures
from cortex.lib.model import (
    ModelsHolder,
    ModelsTree,  # only when num workers = 1
)

from cortex.lib.log import refresh_logger, cx_logger
from cortex.lib.exceptions import CortexException, UserException, UserRuntimeException
from cortex import consts

logger = cx_logger()


class Predictor:
    """
    Class to validate/load the predictor class (PythonPredictor, TensorFlowPredictor, ONNXPredictor).
    Also makes the specified models in cortex.yaml available to the predictor's implementation.
    """

    def __init__(self, provider: str, api_spec: dict, model_dir: str):
        """
        Args:
            provider: "local" or "aws".
            api_spec: API configuration.
            model_dir: Where the models are stored on disk.
        """

        self.provider = provider

        self.type = predictor_type_from_api_spec(api_spec)
        self.path = api_spec["predictor"]["path"]
        self.config = api_spec["predictor"].get("config", {})

        self.api_spec = api_spec
        self.model_dir = model_dir

        self.model_caching = self._is_model_caching_enabled()
        self.multiple_processes = self.api_spec["predictor"]["processes_per_replica"] > 1

        # model caching can only be enabled when processes_per_replica is 1
        # model side-reloading is supported for any number of processes_per_replica

        if self.model_caching:
            self.models = ModelsHolder(
                self.type,
                self.model_dir,
                mem_cache_size=self.api_spec["predictor"]["models"]["cache_size"],
                disk_cache_size=self.api_spec["predictor"]["models"]["disk_cache_size"],
            )
        elif not self.model_caching and self.type not in [
            TensorFlowPredictorType,
            TensorFlowNeuronPredictorType,
        ]:
            self.models = ModelsHolder(self.type, self.model_dir)
        else:
            self.models = None

        if self.multiple_processes or (
            not self.multiple_processes
            and self.type in [TensorFlowPredictorType, TensorFlowNeuronPredictorType]
        ):
            self.models_tree = None
        else:
            self.models_tree = ModelsTree()

        self.crons = []

    def initialize_client(
        self, tf_serving_host: Optional[str] = None, tf_serving_port: Optional[str] = None
    ) -> Union[PythonClient, TensorFlowClient, ONNXClient]:
        """
        Initialize client that gives access to models specified in the API spec (cortex.yaml).

        Args:
            tf_serving_host: Host of TF serving server. To be only used when the TensorFlow predictor is used.
            tf_serving_port: Port of TF serving server. To be only used when the TensorFlow predictor is used.

        Return:
            The client for the respective predictor type.
        """

        signature_message = None
        client = None

        if self.type == PythonPredictorType:
            client = PythonClient(self.api_spec, self.models, self.model_dir, self.models_tree)

        if self.type in [TensorFlowPredictorType, TensorFlowNeuronPredictorType]:
            tf_serving_address = tf_serving_host + ":" + tf_serving_port
            client = TensorFlowClient(
                tf_serving_address, self.api_spec, self.models, self.model_dir, self.models_tree
            )

        if self.type == ONNXPredictorType:
            client = ONNXClient(self.api_spec, self.models, self.model_dir, self.models_tree)

        # show client.input_signatures with logger.info

        return client

    def initialize_impl(
        self, project_dir: str, client: Optional[Union[PythonClient, TensorFlowClient, ONNXClient]]
    ):
        """
        Initialize predictor class as provided by the user.
        """

        # initialize predictor class
        class_impl = self.class_impl(project_dir)
        try:
            if self.type == PythonPredictorType:
                if _are_models_specified(None, self.api_spec):
                    initialized_impl = class_impl(python_client=client, config=self.config)
                    client.set_load_method(initialized_impl.load_model)
                else:
                    initialized_impl = class_impl(config=self.config)
            if self.type in [TensorFlowPredictorType, TensorFlowNeuronPredictorType]:
                initialized_impl = class_impl(tensorflow_client=client, config=self.config)
            if self.type == ONNXPredictorType:
                initialized_impl = class_impl(onnx_client=client, config=self.config)
        except Exception as e:
            raise UserRuntimeException(self.path, "__init__", str(e)) from e
        finally:
            refresh_logger()

        # initialize the crons
        if self.multiple_processes and self.type not in [
            TensorFlowPredictorType,
            TensorFlowNeuronPredictorType,
        ]:
            self.crons = [
                FileBasedModelsTreeUpdater(
                    interval=10, api_spec=self.api_spec, download_dir=self.model_dir
                )
            ]
        elif self.multiple_processes and self.type in [
            TensorFlowPredictorType,
            TensorFlowNeuronPredictorType,
        ]:
            self.crons = [
                TFSModelLoader(
                    interval=10,
                    api_spec=self.api_spec,
                    address=client.tf_serving_url,
                    tfs_model_dir=self.model_dir,
                    download_dir=self.model_dir,
                )
            ]
        else:
            self.crons = [
                ModelTreeUpdater(
                    interval=10,
                    api_spec=self.api_spec,
                    tree=self.models_tree,
                    ondisk_models_dir=self.model_dir,
                ),
                ModelsGC(
                    interval=10,
                    api_spec=self.api_spec,
                    models=self.models,
                    tree=self.models_tree,
                ),
            ]
            if self.model_caching:
                self.crons.append(
                    ModelPreloader(
                        interval=10,
                        caching=self.model_caching,
                        models=self.models,
                        tree=self.models_tree,
                    ),
                )

        # start the crons
        for cron in self.crons:
            cron.start()

        return initialized_impl

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

    def _is_model_caching_enabled(self) -> bool:
        """
        Checks if model caching is enabled (models:cache_size and models:disk_cache_size).
        """
        if (
            self.api_spec["predictor"]["models"]["cache_size"] is not None
            and self.api_spec["predictor"]["models"]["disk_cache_size"] is not None
        ):
            return True
        return False

    def __del__(self) -> None:
        for cron in self.crons:
            cron.stop()
        for cron in self.crons:
            cron.join()


def _are_models_specified(impl: Any, api_spec: dict) -> bool:
    """
    Checks if models have been specified when the API spec (cortex.yaml).

    Args:
        impl: Dummy argument for the predictor validation.
        api_spec: API configuration.
    """
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
            "condition_args": {"python_client": _are_models_specified},
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
            "condition": _are_models_specified,
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
