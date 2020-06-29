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

import onnxruntime as rt
import numpy as np

from cortex.lib.log import cx_logger
from cortex.lib import util
from cortex.lib.exceptions import UserRuntimeException, CortexException, UserException
from cortex.lib.type.model import Model, get_model_names
from cortex import consts


class ONNXClient:
    def __init__(self, models):
        """Setup ONNX runtime session.

        Args:
            models ([Model]): List of models deployed with ONNX container.
        """
        self._models = models
        self._model_names = get_model_names(models)

        self._sessions = {}
        self._signatures = {}
        self._input_signatures = {}
        for model in models:
            self._sessions[model.name] = rt.InferenceSession(model.base_path)
            self._signatures[model.name] = self._sessions[model.name].get_inputs()

            metadata = {}
            for meta in self._signatures[model.name]:
                numpy_type = ONNX_TO_NP_TYPE.get(meta.type, meta.type)
                metadata[meta.name] = {"shape": meta.shape, "type": numpy_type}
            self._input_signatures[model.name] = metadata

    def predict(self, model_input, model_name=None):
        """Validate input, convert it to a dictionary of input_name to numpy.ndarray, and make a prediction.

        Args:
            model_input: Input to the model.
            model_name: Model to use when multiple models are deployed in a single API.

        Returns:
            numpy.ndarray: The prediction returned from the model.
        """
        if consts.SINGLE_MODEL_NAME in self._model_names:
            return self._run_inference(model_input, consts.SINGLE_MODEL_NAME)

        if model_name is None:
            raise UserRuntimeException(
                "model_name was not specified, choose one of the following: {}".format(
                    self._model_names
                )
            )

        if model_name not in self._model_names:
            raise UserRuntimeException(
                "'{}' model wasn't found in the list of available models: {}".format(
                    model_name, self._model_names
                )
            )

        return self._run_inference(model_input, model_name)

    def _run_inference(self, model_input, model_name):
        input_dict = convert_to_onnx_input(model_input, self._signatures[model_name], model_name)
        return self._sessions[model_name].run([], input_dict)

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
