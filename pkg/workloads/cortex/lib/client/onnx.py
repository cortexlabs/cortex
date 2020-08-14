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
from cortex.lib.exceptions import UserRuntimeException, CortexException, UserException
from cortex.lib.model import ModelsTree, CuratedModelResources, find_ondisk_model_versions
from cortex.lib.storage import LockedFile
from cortex import consts

logger = cx_logger()


class ONNXClient:
    def __init__(
        self, models: ModelsTree, model_dir: str, api_spec: dict, lock_dir: str = "/run/cron"
    ):
        """
        Setup ONNX runtime session.

        Args:
            model_dir: Where the models are saved on disk.
            api_spec: API configuration.
            lock_dir: Where the resource locks are found.
        """

        self._models = models
        self._model_dir = model_dir
        self._api_spec = api_spec
        self._lock_dir = lock_dir

        self._spec_models = CuratedModelResources(api_spec["curated_model_resources"])

        if self._api_spec["predictor"]["models"]["dir"] is not None:
            self._models_dir = True
        else:
            self._models_dir = False
            self._spec_model_names = self._spec_models.get_field("name")

    def predict(self, model_input: Any, model_name: str = None, model_version: str = None):
        """
        Validate input, convert it to a dictionary of input_name to numpy.ndarray, and make a prediction.

        Args:
            model_input: Input to the model.
            model_name: Model to use when multiple models are deployed in a single API.
            model_version: Model version to use. If not specified, it will default to the latest version.

        Returns:
            The prediction returned from the model.
        """

        # when predictor:model_path or predictor:models:paths is specified
        if self._models_dir is None:

            # when predictor:model_path is provided
            if consts.SINGLE_MODEL_NAME in self._spec_model_names:
                if model_version is None:
                    model_version = self.get_highest_model_version(consts.SINGLE_MODEL_NAME)
                return self._run_inference(model_input, consts.SINGLE_MODEL_NAME, version)

            if model_name is None:
                raise UserRuntimeException(
                    f"model_name was not specified, choose one of the following: {self._spec_model_names}"
                )

            if model_name not in self._spec_model_names:
                raise UserRuntimeException(
                    f"'{model_name}' model wasn't found in the list of available models"
                )

            # when predictor:models:paths is specified
            if model_version is None:
                model_version = self.get_highest_model_version(model_name)
            return self._run_inference(model_input, consts.SINGLE_MODEL_NAME, version)

        # when predictor:models:dir is specified
        if self._models_dir:
            pass

    def _run_inference(self, model_input: Any, model_name: str, model_version: str) -> Any:
        """
        Run the inference on model model_name of version model_version.
        """

        model = self._models.get_model(model_name, model_version)
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
            RuntimeException: if another thread to load the model at the very same time.

        Returns:
            The model as returned by self._load_model method.
            None if the model wasn't found.
        """

        found_model = True
        resource = model_name + "-" + model_version

        with LockedFile(resource, "r", reader_lock=True) as f:
            status = f.read()
            if status == "" or status == "not available":
                found_model = False
            else:
                if not self._models.has_model(model_name, model_version):
                    model_path = os.path.join(self._model_dir, model_name, model_version)
                    model = self._load_model(model_path)
                    self._models.load_model(model, model_name, model_version)
                else:
                    model = self._get_model(model_name, model_version)

        if found_model:
            return model
        return None

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

    def get_highest_model_version(self, model_name: str) -> str:
        versions = find_ondisk_model_versions(self._lock_dir, model_name)
        if len(versions) == 0:
            raise UserRuntimeException(
                "'{}' model's versions have been removed; add at least a version to the model to resume operations".format(
                    model_name
                )
            )

        highest = max(versions)
        return highest

    @property
    def sessions(self):
        return self._sessions

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
