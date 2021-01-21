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

import collections


class PredictorType(collections.namedtuple("PredictorType", "type")):
    def __str__(self) -> str:
        return str(self.type)

    def __repr__(self) -> str:
        return str(self.type)


PythonPredictorType = PredictorType("python")

TensorFlowPredictorType = PredictorType("tensorflow")
TensorFlowNeuronPredictorType = PredictorType("tensorflow-neuron")

ONNXPredictorType = PredictorType("onnx")


def predictor_type_from_string(predictor_type: str) -> PredictorType:
    """
    Get predictor type from string.

    Args:
        predictor_type: "python", "tensorflow", "onnx" or "tensorflow-neuron"

    Raises:
        ValueError if predictor_type does not hold the right value.
    """
    predictor_types = [
        PythonPredictorType,
        TensorFlowPredictorType,
        ONNXPredictorType,
        TensorFlowNeuronPredictorType,
    ]
    for candidate in predictor_types:
        if str(candidate) == predictor_type:
            return candidate
    raise ValueError(
        "predictor_type can only be 'python', 'tensorflow', 'onnx' or 'tensorflow-neuron'"
    )


def predictor_type_from_api_spec(api_spec: dict) -> PredictorType:
    """
    Get predictor type from API spec.
    """
    if api_spec["compute"]["inf"] > 0 and api_spec["predictor"]["type"] == str(
        TensorFlowPredictorType
    ):
        return predictor_type_from_string("tensorflow-neuron")
    return predictor_type_from_string(api_spec["predictor"]["type"])
