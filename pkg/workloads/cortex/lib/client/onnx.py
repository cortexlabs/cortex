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
import onnxruntime as rt
import numpy as np

from typing import Any

from cortex.lib.log import cx_logger
from cortex.lib import util
from cortex.lib.exceptions import UserRuntimeException, CortexException, UserException, WithBreak
from cortex.lib.model import (
    ModelsHolder,
    LockedGlobalModelsGC,
    LockedModel,
    ModelsTree,
    LockedModelsTree,
    CuratedModelResources,
    find_ondisk_model_info,
    find_ondisk_models,
)
from cortex.lib.concurrency import LockedFile
from cortex import consts

logger = cx_logger()


class ONNXClient:
    def __init__(
        self,
        api_spec: dict,
        models: ModelsHolder,
        models_tree: ModelsTree,
        model_dir: str,
        lock_dir: str = "/run/cron",
    ):
        """
        Setup ONNX runtime session.

        Args:
            api_spec: API configuration.

            models: Holding all models into memory.
            models_tree: A tree of the available models from upstream. Only when caching is enabled. 
            model_dir: Where the models are saved on disk.
            
            lock_dir: Where the resource locks are found. Only when caching is disabled.
        """

        self._api_spec = api_spec
        self._models = models
        self._models_tree = models_tree
        self._model_dir = model_dir
        self._lock_dir = lock_dir

        self._spec_models = CuratedModelResources(api_spec["curated_model_resources"])

        if self._api_spec["predictor"]["models"]["dir"] is not None:
            self._models_dir = True
        else:
            self._models_dir = False
            self._spec_model_names = self._spec_models.get_field("name")

        if (
            self._api_spec["predictor"]["models"]["cache_size"] is not None
            and self._api_spec["predictor"]["models"]["disk_cache_size"] is not None
        ):
            self._cache_enabled = True
        else:
            self._cache_enabled = False

        self._models.set_callback("load", self._load_model)

    # TODO use "latest" for model version instead
    def predict(self, model_input: Any, model_name: str = None, model_version: str = "highest"):
        """
        Validate input, convert it to a dictionary of input_name to numpy.ndarray, and make a prediction.

        Args:
            model_input: Input to the model.
            model_name: Model to use when multiple models are deployed in a single API.
            model_version: Model version to use. Can also be "highest" for picking the highest version or "latest" for picking the most recent version. 

        Returns:
            The prediction returned from the model.
        """

        if model_version not in ["highest", "latest"] or not model_version.isnumeric():
            raise UserRuntimeException(
                "model_version must be either a parse-able numeric value or 'highest' or 'latest'"
            )

        # when predictor:model_path or predictor:models:paths is specified
        if self._models_dir is None:

            # when predictor:model_path is provided
            if consts.SINGLE_MODEL_NAME in self._spec_model_names:
                return self._run_inference(model_input, consts.SINGLE_MODEL_NAME, model_version)

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
            if not self._cache_enabled:
                available_models = find_ondisk_models(self._lock_dir)
                if model_name not in available_models:
                    raise UserRuntimeException(
                        f"'{model_name}' model wasn't found in the list of available models"
                    )

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
        input_dict = convert_to_onnx_input(model_input, model["signatures"], model_name)
        return model["session"].run([], input_dict)

    def _get_model(self, model_name: str, model_version: str) -> Any:
        """
        Checks if versioned model is on disk, then checks if model is in memory,
        and if not, it loads it into memory, and returns the model.

        Args:
            model_name: Name of the model, as it's specified in predictor:models:paths or in the other case as they are called on disk.
            model_version: Version of the model, as they are found on disk.

        Exceptions:
            RuntimeError: if another thread to load the model at the very same time.

        Returns:
            The model as returned by self._load_model method.
            None if the model wasn't found or if it didn't pass the validation.
        """

        model = None
        tag = ""
        tags = []
        if model_version in ["latest", "highest"]:
            tag = model_version
            tags = ["latest", "highest"]

        if not self._cache_enabled:
            # determine model version
            if tag != "":
                model_version = self._get_model_version_from_disk(model_name, tag)
            model_id = model_name + "-" + model_version

            # grab shared access to versioned model
            with LockedFile(model_id, "r", reader_lock=True) as f:

                # check model status
                file_status = f.read()
                if file_status == "" or file_status == "not available":
                    raise WithBreak()

                current_upstream_ts = int(file_status.split(" ")[1])
                update_model = False

                # grab shared access to models holder and retrieve model
                with LockedModel(self._models, "r", model_name, model_version):
                    status, upstream_ts = self._models.has_model(model_name, model_version)
                    if status == "not-available" or (
                        status == "in-memory" and upstream_ts < current_upstream_ts
                    ):
                        update_model = True
                        raise WithBreak()
                    model, _ = self._models.get_model(model_name, model_version, tag)

                # load model into memory and retrieve it
                if update_model:
                    with LockedModel(self._models, "w", model_name, model_version):
                        status, _ = self._models.has_model(model_name, model_version)
                        if status == "not-available":
                            disk_path = os.path.join(self._model_dir, model_name, model_version)
                            self._models.load_model(
                                model_name, model_version, disk_path, current_upstream_ts, tags,
                            )
                        else:
                            model, _ = self._models.get_model(model_name, model_version, tag)

        if self._cache_enabled:
            # determine model version
            try:
                if tag != "":
                    model_version = self._get_model_version_from_tree(
                        model_name, tag, self._models_tree.model_info(model_name)
                    )
            except ValueError:
                # if model_name hasn't been found
                return None

            # grab shared access to model tree
            available_model = True
            with LockedModelsTree(self._models_tree, "r", model_name, model_version):

                # check if the versioned model exists
                model_id = model_name + "-" + model_version
                if model_id not in self._models_tree:
                    available_model = False
                    raise WithBreak()

                # retrieve model tree's metadata
                upstream_model = self._models_tree[model_id]
                current_upstream_ts = upstream_model["timestamp"]

            if not available_model:
                return None

            # grab shared access to models holder and retrieve model
            update_model = False
            with LockedModel(self._models, "r", model_name, model_version):
                status, upstream_ts = self._models.has_model(model_name, model_version)
                if status in ["not-available", "on-disk"] or (
                    status != "not-available" and upstream_ts < current_upstream_ts
                ):
                    update_model = True
                    raise WithBreak()
                model, _ = self._models.get_model(model_name, model_version, tag)

            # download, load into memory the model and retrieve it
            if update_model:
                # grab exclusive access to models holder
                with LockedModel(self._models, "w", model_name, model_version):

                    # check model status
                    status, _ = self._models.has_model(model_name, model_version)

                    # download model
                    if status == "not-available":
                        date = self._models.download_model(
                            upstream_model["bucket"],
                            model_name,
                            model_version,
                            upstream_model["path"],
                        )
                        if not date:
                            raise WithBreak()
                        current_upstream_ts = date.timestamp()

                    # load model
                    disk_path = os.path.join(self._model_dir, model_name, model_version)
                    try:
                        self._models.load_model(
                            model_name, model_version, disk_path, current_upstream_ts, tags,
                        )
                    except Exception:
                        raise WithBreak()

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

        model_path = os.listdir(model_path)[0]
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

    def _get_model_version_from_disk(self, model_name: str, tag: str) -> str:
        """
        Get the version for a specific model name based on the version tag - either "latest" or "highest".
        Must only be used when caching is disabled.
        """

        if tag not in ["latest", "highest"]:
            raise ValueError("invalid tag; must be either 'latest' or 'highest'")

        versions, timestamps = find_ondisk_model_info(self._lock_dir, model_name)
        if len(versions) == 0:
            raise UserRuntimeException(
                "'{}' model's versions have been removed; add at least a version to the model to resume operations".format(
                    model_name
                )
            )

        if tag == "latest":
            index = timestamps.index(max(timestamps))
            return versions[index]
        else:
            return max(versions)

    def _get_model_version_from_tree(self, model_name: str, tag: str, model_info: dict) -> str:
        """
        Get the version for a specific model name based on the version tag - either "latest" or "highest".
        Must only be used when caching is enabled.
        """

        if tag not in ["latest", "highest"]:
            raise ValueError("invalid tag; must be either 'latest' or 'highest'")

        versions, timestamps = model_info["versions"], model_info["timestamps"]
        if tag == "latest":
            index = timestamps.index(max(timestamps))
            return versions[index]
        else:
            return max(versions)

    # TODO retrieve sessions properly for cortex get
    @property
    def sessions(self):
        return self._sessions

    # TODO retrieve input_signatures properly for cortex get
    @property
    def input_signatures(self):
        return self._input_signatures


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
