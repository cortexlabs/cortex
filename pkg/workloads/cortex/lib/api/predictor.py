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
import shutil
import datetime
import glob
from copy import deepcopy
from typing import Any, Optional, Union

# types
from cortex.lib.type import (
    predictor_type_from_api_spec,
    PredictorType,
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
    FileBasedModelsGC,
    TFSAPIServingThreadUpdater,
    ModelsGC,
    ModelTreeUpdater,
    ModelPreloader,
)

# structures
from cortex.lib.model import (
    ModelsHolder,
    ModelsTree,  # only when num workers = 1
)

# model validation
from cortex.lib.model import validate_model_paths

# misc
from cortex.lib.storage import S3
from cortex.lib import util
from cortex.lib.log import refresh_logger, cx_logger as logger
from cortex.lib.exceptions import CortexException, UserException, UserRuntimeException
from cortex import consts


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

        self.crons = []
        if not _are_models_specified(None, self.api_spec):
            return

        self.model_dir = model_dir

        self.caching_enabled = self._is_model_caching_enabled()
        self.multiple_processes = self.api_spec["predictor"]["processes_per_replica"] > 1

        # model caching can only be enabled when processes_per_replica is 1
        # model side-reloading is supported for any number of processes_per_replica

        if self.caching_enabled:
            self.models = ModelsHolder(
                self.type,
                self.model_dir,
                mem_cache_size=self.api_spec["predictor"]["models"]["cache_size"],
                disk_cache_size=self.api_spec["predictor"]["models"]["disk_cache_size"],
                on_download_callback=model_downloader,
            )
        elif not self.caching_enabled and self.type not in [
            TensorFlowPredictorType,
            TensorFlowNeuronPredictorType,
        ]:
            self.models = ModelsHolder(self.type, self.model_dir)
        else:
            self.models = None

        if self.multiple_processes:
            self.models_tree = None
        else:
            self.models_tree = ModelsTree()

    def initialize_client(
        self, tf_serving_host: Optional[str] = None, tf_serving_port: Optional[str] = None
    ) -> Union[PythonClient, TensorFlowClient, ONNXClient]:
        """
        Initialize client that gives access to models specified in the API spec (cortex.yaml).
        Only applies when models are provided in the API spec.

        Args:
            tf_serving_host: Host of TF serving server. To be only used when the TensorFlow predictor is used.
            tf_serving_port: Port of TF serving server. To be only used when the TensorFlow predictor is used.

        Return:
            The client for the respective predictor type.
        """

        signature_message = None
        client = None

        if _are_models_specified(None, self.api_spec):
            if self.type == PythonPredictorType:
                client = PythonClient(self.api_spec, self.models, self.model_dir, self.models_tree)
                if not self.caching_enabled:
                    cron = FileBasedModelsGC(
                        interval=10, models=self.models, download_dir=self.model_dir
                    )
                    cron.start()

            if self.type in [TensorFlowPredictorType, TensorFlowNeuronPredictorType]:
                tf_serving_address = tf_serving_host + ":" + tf_serving_port
                client = TensorFlowClient(
                    tf_serving_address,
                    self.api_spec,
                    self.models,
                    self.model_dir,
                    self.models_tree,
                )
                if not self.caching_enabled:
                    cron = TFSAPIServingThreadUpdater(interval=2.5, client=client._client)
                    cron.start()

            if self.type == ONNXPredictorType:
                client = ONNXClient(self.api_spec, self.models, self.model_dir, self.models_tree)

        return client

    def initialize_impl(
        self,
        project_dir: str,
        client: Union[Union[PythonClient, TensorFlowClient, ONNXClient]],
        job_spec: Optional[dict] = None,
    ):
        """
        Initialize predictor class as provided by the user.

        job_spec is a dictionary when the "kind" of the API is set to "BatchAPI". Otherwise, it's None.
        """

        # build args
        class_impl = self.class_impl(project_dir)
        constructor_args = inspect.getfullargspec(class_impl.__init__).args
        config = deepcopy(self.config)
        args = {}
        if job_spec is not None and job_spec.get("config") is not None:
            util.merge_dicts_in_place_overwrite(config, job_spec["config"])
        if "config" in constructor_args:
            args["config"] = config
        if "job_spec" in constructor_args:
            args["job_spec"] = job_spec

        # initialize predictor class
        try:
            if self.type == PythonPredictorType:
                if _are_models_specified(None, self.api_spec):
                    args["python_client"] = client
                    initialized_impl = class_impl(**args)
                    client.set_load_method(initialized_impl.load_model)
                else:
                    initialized_impl = class_impl(**args)
            if self.type in [TensorFlowPredictorType, TensorFlowNeuronPredictorType]:
                args["tensorflow_client"] = client
                initialized_impl = class_impl(**args)
            if self.type == ONNXPredictorType:
                args["onnx_client"] = client
                initialized_impl = class_impl(**args)
        except Exception as e:
            raise UserRuntimeException(self.path, "__init__", str(e)) from e
        finally:
            refresh_logger()

        # initialize the crons if models have been specified
        if _are_models_specified(None, self.api_spec):
            if not self.multiple_processes and self.caching_enabled:
                self.crons += [
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
                    # ModelPreloader(
                    #     interval=10,
                    #     caching=self.caching_enabled,
                    #     models=self.models,
                    #     tree=self.models_tree,
                    # ),
                ]

            if not self.caching_enabled and self.type in [PythonPredictorType, ONNXPredictorType]:
                self.crons += [
                    FileBasedModelsGC(interval=10, models=self.models, download_dir=self.model_dir)
                ]

        for cron in self.crons:
            cron.start()

        return initialized_impl

    def class_impl(self, project_dir):
        if self.type in [TensorFlowPredictorType, TensorFlowNeuronPredictorType]:
            target_class_name = "TensorFlowPredictor"
            validations = TENSORFLOW_CLASS_VALIDATION
        elif self.type == ONNXPredictorType:
            target_class_name = "ONNXPredictor"
            validations = ONNX_CLASS_VALIDATION
        elif self.type == PythonPredictorType:
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
                            f"multiple definitions for {target_class_name} class found; please check your imports and class definitions and ensure that there is only one Predictor class definition"
                        )
                    predictor_class = class_df[1]
            if predictor_class is None:
                raise UserException(f"{target_class_name} class is not defined")
            _validate_impl(predictor_class, validations, self.api_spec)
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
            self.api_spec["predictor"]["models"]
            and self.api_spec["predictor"]["models"]["cache_size"] is not None
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
    Checks if models have been specified in the API spec (cortex.yaml).

    Args:
        impl: Dummy argument for the predictor validation.
        api_spec: API configuration.
    """
    if api_spec["predictor"]["model_path"] is not None:
        return True

    if api_spec["predictor"]["models"] and (
        api_spec["predictor"]["models"]["dir"] is not None
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
        _validate_optional_fn_args(impl, optional_func_signature, api_spec)

    for required_func_signature in impl_req.get("required", []):
        _validate_required_fn_args(impl, required_func_signature, api_spec)

    for required_func_signature in impl_req.get("conditional", []):
        if required_func_signature["condition"](impl, api_spec):
            _validate_required_fn_args(impl, required_func_signature, api_spec)


def _validate_optional_fn_args(impl, func_signature, api_spec):
    if getattr(impl, func_signature["name"], None):
        _validate_required_fn_args(impl, func_signature, api_spec)


def _validate_required_fn_args(impl, func_signature, api_spec):
    target_class_name = impl.__name__

    fn = getattr(impl, func_signature["name"], None)
    if not fn:
        raise UserException(
            f"class {target_class_name}",
            f'required method "{func_signature["name"]}" is not defined',
        )

    if not callable(fn):
        raise UserException(
            f"class {target_class_name}",
            f'"{func_signature["name"]}" is defined, but is not a method',
        )

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
                f"class {target_class_name}",
                f'invalid signature for method "{fn_str}"',
                f'"{arg_name}" is a required argument, but was not provided',
            )

        if arg_name == "self":
            if argspec.args[0] != "self":
                raise UserException(
                    f"class {target_class_name}",
                    f'invalid signature for method "{fn_str}"',
                    f'"self" must be the first argument',
                )

    seen_args = []
    for arg_name in argspec.args:
        if arg_name not in required_args and arg_name not in optional_args:
            raise UserException(
                f"class {target_class_name}",
                f'invalid signature for method "{fn_str}"',
                f'"{arg_name}" is not a supported argument',
            )

        if arg_name in seen_args:
            raise UserException(
                f"class {target_class_name}",
                f'invalid signature for method "{fn_str}"',
                f'"{arg_name}" is duplicated',
            )

        seen_args.append(arg_name)


def model_downloader(
    predictor_type: PredictorType,
    bucket_name: str,
    model_name: str,
    model_version: str,
    model_path: str,
    temp_dir: str,
    model_dir: str,
) -> Optional[datetime.datetime]:
    """
    Downloads model to disk. Validates the S3 model path and the downloaded model as well.

    Args:
        bucket_name: Name of the bucket where the model is stored.
        model_name: Name of the model. Is part of the model's local path.
        model_version: Version of the model. Is part of the model's local path.
        model_path: S3 model prefix to the versioned model.
        temp_dir: Where to temporarily store the model for validation.
        model_dir: The top directory of where all models are stored locally.

    Returns:
        The model's timestamp. None if the model didn't pass the validation, if it doesn't exist or if there are not enough permissions.
    """

    logger().info(
        f"downloading from bucket {bucket_name}/{model_path}, model {model_name} of version {model_version}, temporarily to {temp_dir} and then finally to {model_dir}"
    )

    s3_client = S3(bucket_name, client_config={})

    # validate upstream S3 model
    sub_paths, ts = s3_client.search(model_path)
    try:
        validate_model_paths(sub_paths, predictor_type, model_path)
    except CortexException:
        logger().info(f"failed validating {model_name} {model_version}")
        return None

    # download model to temp dir
    temp_dest = os.path.join(temp_dir, model_name, model_version)
    try:
        s3_client.download_dir_contents(model_path, temp_dest)
    except CortexException:
        logger().info(f"failed downloading {model_name} {model_version} to temp dir {temp_dest}")
        shutil.rmtree(temp_dest)
        return None

    # validate model
    model_contents = glob.glob(temp_dest + "*/**", recursive=True)
    model_contents = util.remove_non_empty_directory_paths(model_contents)
    try:
        validate_model_paths(model_contents, predictor_type, temp_dest)
    except CortexException:
        logger().info(f"failed validating {model_name} {model_version} from temp dir")
        shutil.rmtree(temp_dest)
        return None

    # move model to dest dir
    model_top_dir = os.path.join(model_dir, model_name)
    ondisk_model_version = os.path.join(model_top_dir, model_version)
    logger().info(f"moving {model_name} {model_version} to final dir {ondisk_model_version}")
    if os.path.isdir(ondisk_model_version):
        shutil.rmtree(ondisk_model_version)
    shutil.move(temp_dest, ondisk_model_version)

    return max(ts)
