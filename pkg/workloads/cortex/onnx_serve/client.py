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

import onnxruntime as rt
import numpy as np

from cortex.lib.log import cx_logger
from cortex.lib import util
from cortex.lib.exceptions import CortexException, UserException


class ONNXClient:
    def __init__(self, model_path):
        """Setup ONNX runtime session.

        Args:
            model_path (string): Path to model in local file system.
        """
        self._model_path = model_path
        session = rt.InferenceSession(model_path)

        self._session = session
        self._signature = session.get_inputs()
        metadata = {}
        for meta in self._signature:
            numpy_type = ONNX_TO_NP_TYPE.get(meta.type, meta.type)
            metadata[meta.name] = {"shape": meta.shape, "type": numpy_type}

        self._input_signature = metadata

    def predict(self, payload):
        """Validate payload, convert payload to a dictionary of input_name to numpy.ndarray and make a prediction.

        Args:
            payload: Input to model

        Returns:
            numpy.ndarray: Prediction
        """
        inference_input = convert_to_onnx_input(payload, self._signature)
        model_output = self._session.run([], inference_input)
        return model_output

    @property
    def session(self):
        return self._session

    @property
    def input_signature(self):
        return self._input_signature


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


def transform_to_numpy(input_pyobj, input_metadata):
    target_dtype = ONNX_TO_NP_TYPE[input_metadata.type]
    target_shape = input_metadata.shape

    try:
        for idx, dim in enumerate(target_shape):
            if type(dim) is not int:
                target_shape[idx] = 1

        if type(input_pyobj) is np.ndarray:
            np_arr = input_pyobj
            if np.issubdtype(np_arr.dtype, np.number) == np.issubdtype(target_dtype, np.number):
                if str(np_arr.dtype) != target_dtype:
                    np_arr = np_arr.astype(target_dtype)
            else:
                raise ValueError(
                    "expected dtype '{}' but found '{}'".format(target_dtype, np_arr.dtype)
                )
        else:
            np_arr = np.array(input_pyobj, dtype=target_dtype)
        np_arr = np_arr.reshape(target_shape)
        return np_arr
    except Exception as e:
        raise UserException("failed to convert to numpy array", str(e)) from e


def convert_to_onnx_input(payload, input_metadata_list):
    input_dict = {}
    if len(input_metadata_list) == 1:
        input_metadata = input_metadata_list[0]
        if util.is_dict(payload):
            if payload.get(input_metadata.name) is None:
                raise UserException('missing key "{}"'.format(input_metadata.name))
            input_dict[input_metadata.name] = transform_to_numpy(
                payload[input_metadata.name], input_metadata
            )
        else:
            try:
                input_dict[input_metadata.name] = transform_to_numpy(payload, input_metadata)
            except CortexException as e:
                e.wrap('key "{}"'.format(input_metadata.name))
                raise
    else:
        for input_metadata in input_metadata_list:
            if not util.is_dict(payload):
                expected_keys = [metadata.name for metadata in input_metadata_list]
                raise UserException(
                    "expected payload to be a dictionary with keys {}".format(
                        ", ".join('"' + key + '"' for key in expected_keys)
                    )
                )

            if payload.get(input_metadata.name) is None:
                raise UserException('missing key "{}"'.format(input_metadata.name))
            try:
                input_dict[input_metadata.name] = transform_to_numpy(
                    payload[input_metadata.name], input_metadata
                )
            except CortexException as e:
                e.wrap('key "{}"'.format(input_metadata.name))
                raise
    return input_dict
