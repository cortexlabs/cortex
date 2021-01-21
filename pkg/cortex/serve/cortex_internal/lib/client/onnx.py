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

import os
import datetime
import threading as td
import multiprocessing as mp
from typing import Any, Tuple, Optional

try:
    import onnxruntime as rt

    onnx_dependencies_installed = True
except ImportError:
    onnx_dependencies_installed = False
import numpy as np

from cortex_internal.lib import util
from cortex_internal.lib.exceptions import (
    UserRuntimeException,
    CortexException,
    UserException,
    WithBreak,
)
from cortex_internal.lib.model import (
    ModelsHolder,
    LockedModel,
    ModelsTree,
    LockedModelsTree,
    get_models_from_api_spec,
    find_ondisk_model_info,
    find_ondisk_models_with_lock,
)
from cortex_internal.lib.concurrency import LockedFile
from cortex_internal import consts
from cortex_internal.lib.log import configure_logger

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


class ONNXClient:
    def __init__(
        self,
        api_spec: dict,
        models: ModelsHolder,
        model_dir: str,
        models_tree: Optional[ModelsTree],
        lock_dir: Optional[str] = "/run/cron",
    ):
        """
        Setup ONNX runtime.

        Args:
            api_spec: API configuration.

            models: Holding all models into memory.
            model_dir: Where the models are saved on disk.

            models_tree: A tree of the available models from upstream.
            lock_dir: Where the resource locks are found. Only when processes_per_replica > 0 and caching disabled.
        """
        if not onnx_dependencies_installed:
            raise NameError("onnx dependencies not installed")

        self._api_spec = api_spec
        self._models = models
        self._models_tree = models_tree
        self._model_dir = model_dir
        self._lock_dir = lock_dir

        self._spec_models = get_models_from_api_spec(api_spec)

        if (
            self._api_spec["predictor"]["models"]
            and self._api_spec["predictor"]["models"]["dir"] is not None
        ):
            self._models_dir = True
        else:
            self._models_dir = False
            self._spec_model_names = self._spec_models.get_field("name")

        # only applicable for ONNX file paths (for ONNX filepaths, it must look as if the models are available locally)
        self._spec_local_model_names = self._spec_models.get_local_model_names()
        self._local_model_ts = int(datetime.datetime.now(datetime.timezone.utc).timestamp())

        self._multiple_processes = self._api_spec["predictor"]["processes_per_replica"] > 1
        self._caching_enabled = self._is_model_caching_enabled()

        self._models.set_callback("load", self._load_model)

    def _validate_model_args(
        self, model_name: Optional[str] = None, model_version: str = "latest"
    ) -> Tuple[str, str]:
        """
        Validate the model name and model version.

        Args:
            model_name: Name of the model.
            model_version: Model version to use. Can also be "latest" for picking the highest version.

        Returns:
            The processed model_name, model_version tuple if they had to go through modification.

        Raises:
            UserRuntimeException if the validation fails.
        """

        if model_version != "latest" and not model_version.isnumeric():
            raise UserRuntimeException(
                "model_version must be either a parse-able numeric value or 'latest'"
            )

        # when predictor:models:path or predictor:models:paths is specified
        if not self._models_dir:

            # when when predictor:models:path is provided
            if consts.SINGLE_MODEL_NAME in self._spec_model_names:
                return consts.SINGLE_MODEL_NAME, model_version

            # when predictor:models:paths is specified
            if model_name is None:
                raise UserRuntimeException(
                    f"model_name was not specified, choose one of the following: {self._spec_model_names}"
                )

            if model_name not in self._spec_model_names:
                raise UserRuntimeException(
                    f"'{model_name}' model wasn't found in the list of available models"
                )

        # when predictor:models:dir is specified
        if self._models_dir:
            if model_name is None:
                raise UserRuntimeException("model_name was not specified")
            if not self._caching_enabled:
                available_models = find_ondisk_models_with_lock(self._lock_dir)
                if model_name not in available_models:
                    raise UserRuntimeException(
                        f"'{model_name}' model wasn't found in the list of available models"
                    )

        return model_name, model_version

    def predict(
        self, model_input: Any, model_name: Optional[str] = None, model_version: str = "latest"
    ) -> Any:
        """
        Validate input, convert it to a dictionary of input_name to numpy.ndarray, and make a prediction.

        Args:
            model_input: Input to the model.
            model_name (optional): Name of the model to retrieve (when multiple models are deployed in an API).
                When predictor.models.paths is specified, model_name should be the name of one of the models listed in the API config.
                When predictor.models.dir is specified, model_name should be the name of a top-level directory in the models dir.
            model_version (string, optional): Version of the model to retrieve. Can be omitted or set to "latest" to select the highest version.

        Returns:
            The prediction returned from the model.

        Raises:
            UserRuntimeException if the validation fails.
        """

        model_name, model_version = self._validate_model_args(model_name, model_version)
        return self._run_inference(model_input, model_name, model_version)

    def _run_inference(self, model_input: Any, model_name: str, model_version: str) -> Any:
        """
        Run the inference on model model_name of version model_version.
        """

        model = self._get_model(model_name, model_version)
        if model is None:
            raise UserRuntimeException(
                f"model {model_name} of version {model_version} wasn't found"
            )

        try:
            input_dict = convert_to_onnx_input(model_input, model["signatures"], model_name)
            return model["session"].run([], input_dict)
        except Exception as e:
            raise UserRuntimeException(
                f"failed inference with model {model_name} of version {model_version}", str(e)
            )

    def get_model(self, model_name: Optional[str] = None, model_version: str = "latest") -> dict:
        """
        Validate input and then return the model loaded into a dictionary.
        The counting of tag calls is recorded with this method (just like with the predict method).

        Args:
            model_name: Model to use when multiple models are deployed in a single API.
            model_version: Model version to use. Can also be "latest" for picking the highest version.

        Returns:
            The model as returned by _load_model method.

        Raises:
            UserRuntimeException if the validation fails.
        """
        model_name, model_version = self._validate_model_args(model_name, model_version)
        model = self._get_model(model_name, model_version)
        if model is None:
            raise UserRuntimeException(
                f"model {model_name} of version {model_version} wasn't found"
            )
        return model

    def _get_model(self, model_name: str, model_version: str) -> Any:
        """
        Checks if versioned model is on disk, then checks if model is in memory,
        and if not, it loads it into memory, and returns the model.

        Args:
            model_name: Name of the model, as it's specified in predictor:models:paths or in the other case as they are named on disk.
            model_version: Version of the model, as it's found on disk. Can also infer the version number from the "latest" version tag.

        Exceptions:
            RuntimeError: if another thread tried to load the model at the very same time.

        Returns:
            The model as returned by self._load_model method.
            None if the model wasn't found or if it didn't pass the validation.
        """

        model = None
        tag = ""
        if model_version == "latest":
            tag = model_version

        if not self._caching_enabled:
            # determine model version
            if tag == "latest":
                model_version = self._get_latest_model_version_from_disk(model_name)
            model_id = model_name + "-" + model_version

            # grab shared access to versioned model
            resource = os.path.join(self._lock_dir, model_id + ".txt")
            with LockedFile(resource, "r", reader_lock=True) as f:

                # check model status
                file_status = f.read()
                if file_status == "" or file_status == "not-available":
                    raise WithBreak

                current_upstream_ts = int(file_status.split(" ")[1])
                update_model = False

                # grab shared access to models holder and retrieve model
                with LockedModel(self._models, "r", model_name, model_version):
                    status, local_ts = self._models.has_model(model_name, model_version)
                    if status == "not-available" or (
                        status == "in-memory" and local_ts != current_upstream_ts
                    ):
                        update_model = True
                        raise WithBreak
                    model, _ = self._models.get_model(model_name, model_version, tag)

                # load model into memory and retrieve it
                if update_model:
                    with LockedModel(self._models, "w", model_name, model_version):
                        status, _ = self._models.has_model(model_name, model_version)
                        if status == "not-available" or (
                            status == "in-memory" and local_ts != current_upstream_ts
                        ):
                            if status == "not-available":
                                logger.info(
                                    f"loading model {model_name} of version {model_version} (thread {td.get_ident()})"
                                )
                            else:
                                logger.info(
                                    f"reloading model {model_name} of version {model_version} (thread {td.get_ident()})"
                                )
                            try:
                                self._models.load_model(
                                    model_name,
                                    model_version,
                                    current_upstream_ts,
                                    [tag],
                                )
                            except Exception as e:
                                raise UserRuntimeException(
                                    f"failed (re-)loading model {model_name} of version {model_version} (thread {td.get_ident()})",
                                    str(e),
                                )
                        model, _ = self._models.get_model(model_name, model_version, tag)

        if not self._multiple_processes and self._caching_enabled:
            # determine model version
            try:
                if tag == "latest":
                    model_version = self._get_latest_model_version_from_tree(
                        model_name, self._models_tree.model_info(model_name)
                    )
            except ValueError:
                # if model_name hasn't been found
                raise UserRuntimeException(
                    f"'{model_name}' model of tag {tag} wasn't found in the list of available models"
                )

            # grab shared access to model tree
            available_model = True
            with LockedModelsTree(self._models_tree, "r", model_name, model_version):

                # check if the versioned model exists
                model_id = model_name + "-" + model_version
                if model_id not in self._models_tree:
                    available_model = False
                    raise WithBreak

                # retrieve model tree's metadata
                upstream_model = self._models_tree[model_id]
                current_upstream_ts = int(upstream_model["timestamp"].timestamp())

            if not available_model:
                return None

            # grab shared access to models holder and retrieve model
            update_model = False
            with LockedModel(self._models, "r", model_name, model_version):
                status, local_ts = self._models.has_model(model_name, model_version)
                if status in ["not-available", "on-disk"] or (
                    status != "not-available"
                    and local_ts != current_upstream_ts
                    and not (status == "in-memory" and model_name in self._spec_local_model_names)
                ):
                    update_model = True
                    raise WithBreak
                model, _ = self._models.get_model(model_name, model_version, tag)

            # download, load into memory the model and retrieve it
            if update_model:
                # grab exclusive access to models holder
                with LockedModel(self._models, "w", model_name, model_version):

                    # check model status
                    status, local_ts = self._models.has_model(model_name, model_version)

                    # refresh disk model
                    if model_name not in self._spec_local_model_names and (
                        status == "not-available"
                        or (status in ["on-disk", "in-memory"] and local_ts != current_upstream_ts)
                    ):
                        if status == "not-available":
                            logger.info(
                                f"model {model_name} of version {model_version} not found locally; continuing with the download..."
                            )
                        elif status == "on-disk":
                            logger.info(
                                f"found newer model {model_name} of vesion {model_version} on the {upstream_model['provider']} upstream than the one on the disk"
                            )
                        else:
                            logger.info(
                                f"found newer model {model_name} of vesion {model_version} on the {upstream_model['provider']} upstream than the one loaded into memory"
                            )

                        # remove model from disk and memory
                        if status == "on-disk":
                            logger.info(
                                f"removing model from disk for model {model_name} of version {model_version}"
                            )
                            self._models.remove_model(model_name, model_version)
                        if status == "in-memory":
                            logger.info(
                                f"removing model from disk and memory for model {model_name} of version {model_version}"
                            )
                            self._models.remove_model(model_name, model_version)

                        # download model
                        logger.info(
                            f"downloading model {model_name} of version {model_version} from the {upstream_model['provider']} upstream"
                        )
                        date = self._models.download_model(
                            upstream_model["provider"],
                            upstream_model["bucket"],
                            model_name,
                            model_version,
                            upstream_model["path"],
                        )
                        if not date:
                            raise WithBreak
                        current_upstream_ts = int(date.timestamp())

                    # give the local model a timestamp initialized at start time
                    if model_name in self._spec_local_model_names:
                        current_upstream_ts = self._local_model_ts

                    # load model
                    try:
                        logger.info(
                            f"loading model {model_name} of version {model_version} into memory"
                        )
                        self._models.load_model(
                            model_name,
                            model_version,
                            current_upstream_ts,
                            [tag],
                        )
                    except Exception as e:
                        raise UserRuntimeException(
                            f"failed (re-)loading model {model_name} of version {model_version} (thread {td.get_ident()})",
                            str(e),
                        )

                    # retrieve model
                    model, _ = self._models.get_model(model_name, model_version, tag)

        return model

    def _load_model(self, model_path: str) -> None:
        """
        Load ONNX model from disk.

        Args:
            model_path: Directory path to a model's version on disk.

        Not thread-safe, so this method cannot be called on its own. Must only be called by self._get_model method.
        """

        model_path = os.path.join(model_path, os.listdir(model_path)[0])
        model = {
            "session": rt.InferenceSession(model_path),
        }
        model["signatures"] = model["session"].get_inputs()
        metadata = {}
        for meta in model["signatures"]:
            numpy_type = ONNX_TO_NP_TYPE.get(meta.type, meta.type)
            metadata[meta.name] = {
                "shape": meta.shape,
                "type": numpy_type,
            }
        model["input_signatures"] = metadata

        return model

    def _get_latest_model_version_from_disk(self, model_name: str) -> str:
        """
        Get the highest version of a specific model name.
        Must only be used when caching disabled and processes_per_replica > 0.
        """
        versions, timestamps = find_ondisk_model_info(self._lock_dir, model_name)
        if len(versions) == 0:
            raise UserRuntimeException(
                "'{}' model's versions have been removed; add at least a version to the model to resume operations".format(
                    model_name
                )
            )
        return str(max(map(lambda x: int(x), versions)))

    def _get_latest_model_version_from_tree(self, model_name: str, model_info: dict) -> str:
        """
        Get the highest version of a specific model name.
        Must only be used when processes_per_replica = 1 and caching is enabled.
        """
        versions, timestamps = model_info["versions"], model_info["timestamps"]
        return str(max(map(lambda x: int(x), versions)))

    def _is_model_caching_enabled(self) -> bool:
        """
        Checks if model caching is enabled (models:cache_size and models:disk_cache_size).
        """
        return (
            self._api_spec["predictor"]["models"]
            and self._api_spec["predictor"]["models"]["cache_size"] is not None
            and self._api_spec["predictor"]["models"]["disk_cache_size"] is not None
        )

    @property
    def metadata(self) -> dict:
        """
        The returned dictionary will be like in the following example:
        {
            ...
            "yolov3": {
                "versions": [
                    "2",
                    "1"
                ],
                "timestamps": [
                    1601668127,
                    1601668127
                ]
            }
            ...
        }
        """
        if not self._caching_enabled:
            return find_ondisk_models_with_lock(self._lock_dir, include_timestamps=True)
        else:
            models_info = self._models_tree.get_all_models_info()
            for model_name in models_info.keys():
                del models_info[model_name]["bucket"]
                del models_info[model_name]["model_paths"]
            return models_info

    @property
    def caching(self) -> bool:
        return self._caching_enabled


# https://github.com/microsoft/onnxruntime/blob/v0.4.0/onnxruntime/python/onnxruntime_pybind_mlvalue.cc
ONNX_TO_NP_TYPE = {
    "tensor(float16)": "float16",
    "tensor(float)": "float32",
    "tensor(double)": "float64",
    "tensor(int32)": "int32",
    "tensor(uint32)": "uint32",
    "tensor(int8)": "int8",
    "tensor(uint8)": "uint8",
    "tensor(int16)": "int16",
    "tensor(uint16)": "uint16",
    "tensor(int64)": "int64",
    "tensor(uint64)": "uint64",
    "tensor(bool)": "bool",
    "tensor(string)": "object",
}


def transform_to_numpy(input_pyobj, input_metadata, model_name):
    target_dtype = ONNX_TO_NP_TYPE[input_metadata.type]
    target_shape = input_metadata.shape

    try:
        for idx, dim in enumerate(target_shape):
            if type(dim) is str:
                target_shape[idx] = -1
            elif type(dim) is not int:
                target_shape[idx] = 1

        if type(input_pyobj) is np.ndarray:
            np_arr = input_pyobj
            if np.issubdtype(np_arr.dtype, np.number) == np.issubdtype(target_dtype, np.number):
                if str(np_arr.dtype) != target_dtype:
                    np_arr = np_arr.astype(target_dtype)
            else:
                raise ValueError(
                    "expected dtype '{}' but found '{}' for model '{}'".format(
                        target_dtype, np_arr.dtype, model_name
                    )
                )
        else:
            np_arr = np.array(input_pyobj, dtype=target_dtype)

        # can only infer the size for up to 1 unknown dimension
        if target_shape.count(-1) <= 1:
            np_arr = np_arr.reshape(target_shape)

        return np_arr
    except Exception as e:
        raise UserException(
            "failed to convert to numpy array for model '{}'".format(model_name), str(e)
        ) from e


def convert_to_onnx_input(model_input, input_metadata_list, model_name):
    input_dict = {}
    if len(input_metadata_list) == 1:
        input_metadata = input_metadata_list[0]
        if util.is_dict(model_input):
            if model_input.get(input_metadata.name) is None:
                raise UserException(
                    "missing key '{}' for model '{}'".format(input_metadata.name, model_name)
                )
            input_dict[input_metadata.name] = transform_to_numpy(
                model_input[input_metadata.name], input_metadata, model_name
            )
        else:
            try:
                input_dict[input_metadata.name] = transform_to_numpy(
                    model_input, input_metadata, model_name
                )
            except CortexException as e:
                e.wrap("key '{}' for model '{}'".format(input_metadata.name, model_name))
                raise
    else:
        for input_metadata in input_metadata_list:
            if not util.is_dict(model_input):
                expected_keys = [metadata.name for metadata in input_metadata_list]
                raise UserException(
                    "expected model_input to be a dictionary with keys '{}' for model '{}'".format(
                        ", ".join('"' + key + '"' for key in expected_keys), model_name
                    )
                )

            if model_input.get(input_metadata.name) is None:
                raise UserException(
                    "missing key '{}' for model '{}'".format(input_metadata.name, model_name)
                )
            try:
                input_dict[input_metadata.name] = transform_to_numpy(
                    model_input[input_metadata.name], input_metadata, model_name
                )
            except CortexException as e:
                e.wrap("key '{}' for model '{}'".format(input_metadata.name, model_name))
                raise
    return input_dict
