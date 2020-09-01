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

import time
import sys
import grpc

import tensorflow as tf
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import get_model_metadata_pb2
from tensorflow_serving.apis import prediction_service_pb2_grpc
from google.protobuf import json_format

from cortex.lib.exceptions import UserRuntimeException, UserException, CortexException
from cortex.lib.log import cx_logger
from cortex.lib.type.model import Model, get_model_signature_map, get_model_names
from cortex import consts


class TensorFlowClient:
    def __init__(self, tf_serving_url, models):
        """Setup gRPC connection to TensorFlow Serving container.

        Args:
            tf_serving_url (string): Localhost URL to TF Serving container.
            models        ([Model]): List of models deployed with TF serving container.
        """
        self._tf_serving_url = tf_serving_url
        self._models = models
        self._model_names = get_model_names(models)

        channel = grpc.insecure_channel(tf_serving_url)
        self._stub = prediction_service_pb2_grpc.PredictionServiceStub(channel)

        self._signatures = get_signature_defs(self._stub, models)
        parsed_signature_keys, parsed_signatures = extract_signatures(
            self._signatures, get_model_signature_map(models)
        )
        self._signature_keys = parsed_signature_keys
        self._input_signatures = parsed_signatures

    def predict(self, model_input, model_name=None):
        """Validate model_input, convert it to a Prediction Proto, and make a request to TensorFlow Serving.

        Args:
            model_input: Input to the model.
            model_name: Model to use when multiple models are deployed in a single API.

        Returns:
            dict: TensorFlow Serving response converted to a dictionary.
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
        input_signature = self._input_signatures[model_name]
        signature = self._signatures[model_name]
        signature_key = self._signature_keys[model_name]

        validate_model_input(input_signature, model_input, model_name)
        prediction_request = create_prediction_request(
            signature, signature_key, model_name, model_input
        )
        response_proto = self._stub.Predict(prediction_request, timeout=300.0)
        return parse_response_proto(response_proto)

    @property
    def stub(self):
        return self._stub

    @property
    def input_signatures(self):
        return self._input_signatures


DTYPE_TO_TF_TYPE = {
    "DT_FLOAT": tf.float32,
    "DT_DOUBLE": tf.float64,
    "DT_INT32": tf.int32,
    "DT_UINT8": tf.uint8,
    "DT_INT16": tf.int16,
    "DT_INT8": tf.int8,
    "DT_STRING": tf.string,
    "DT_COMPLEX64": tf.complex64,
    "DT_INT64": tf.int64,
    "DT_BOOL": tf.bool,
    "DT_QINT8": tf.qint8,
    "DT_QUINT8": tf.quint8,
    "DT_QINT32": tf.qint32,
    "DT_BFLOAT16": tf.bfloat16,
    "DT_QINT16": tf.qint16,
    "DT_QUINT16": tf.quint16,
    "DT_UINT16": tf.uint16,
    "DT_COMPLEX128": tf.complex128,
    "DT_HALF": tf.float16,
    "DT_RESOURCE": tf.resource,
    "DT_VARIANT": tf.variant,
    "DT_UINT32": tf.uint32,
    "DT_UINT64": tf.uint64,
}

DTYPE_TO_VALUE_KEY = {
    "DT_INT32": "intVal",
    "DT_INT64": "int64Val",
    "DT_FLOAT": "floatVal",
    "DT_STRING": "stringVal",
    "DT_BOOL": "boolVal",
    "DT_DOUBLE": "doubleVal",
    "DT_HALF": "halfVal",
    "DT_COMPLEX64": "scomplexVal",
    "DT_COMPLEX128": "dcomplexVal",
}


def get_signature_defs(stub, models):
    sigmaps = {}
    for model in models:
        sigmaps[model.name] = get_signature_def(stub, model)

    return sigmaps


def get_signature_def(stub, model):
    limit = 2
    for i in range(limit):
        try:
            request = create_get_model_metadata_request(model.name)
            resp = stub.GetModelMetadata(request, timeout=10.0)
            sigAny = resp.metadata["signature_def"]
            signature_def_map = get_model_metadata_pb2.SignatureDefMap()
            sigAny.Unpack(signature_def_map)
            sigmap = json_format.MessageToDict(signature_def_map)
            return sigmap["signatureDef"]
        except Exception as e:
            print(e)
            cx_logger().warn(
                "unable to read model metadata for model '{}' - retrying ...".format(model.name)
            )

        time.sleep(5)

    raise CortexException(
        "timeout: unable to read model metadata for model '{}'".format(model.name)
    )


def create_get_model_metadata_request(model_name):
    get_model_metadata_request = get_model_metadata_pb2.GetModelMetadataRequest()
    get_model_metadata_request.model_spec.name = model_name
    get_model_metadata_request.metadata_field.append("signature_def")
    return get_model_metadata_request


def extract_signatures(signature_defs, signature_keys):
    parsed_signature_keys = {}
    parsed_signatures = {}
    for model_name in signature_defs:
        parsed_signature_key, parsed_signature = extract_signature(
            signature_defs[model_name],
            signature_keys[model_name],
            model_name,
        )
        parsed_signature_keys[model_name] = parsed_signature_key
        parsed_signatures[model_name] = parsed_signature

    return parsed_signature_keys, parsed_signatures


def extract_signature(signature_def, signature_key, model_name):
    cx_logger().info("signature defs found in model '{}': {}".format(model_name, signature_def))

    available_keys = list(signature_def.keys())
    if len(available_keys) == 0:
        raise UserException("unable to find signature defs in model '{}'".format(model_name))

    if signature_key is None:
        if len(available_keys) == 1:
            cx_logger().info(
                "signature_key was not configured by user, using signature key '{}' for model '{}' (found in the signature def map)".format(
                    available_keys[0],
                    model_name,
                )
            )
            signature_key = available_keys[0]
        elif "predict" in signature_def:
            cx_logger().info(
                "signature_key was not configured by user, using signature key 'predict' for model '{}' (found in the signature def map)".format(
                    model_name
                )
            )
            signature_key = "predict"
        else:
            raise UserException(
                "signature_key was not configured by user, please specify one the following keys '{}' for model '{}' (found in the signature def map)".format(
                    ", ".join(available_keys), model_name
                )
            )
    else:
        if signature_def.get(signature_key) is None:
            possibilities_str = "key: '{}'".format(available_keys[0])
            if len(available_keys) > 1:
                possibilities_str = "keys: '{}'".format("', '".join(available_keys))

            raise UserException(
                "signature_key '{}' was not found in signature def map for model '{}', but found the following {}".format(
                    signature_key, model_name, possibilities_str
                )
            )

    signature_def_val = signature_def.get(signature_key)

    if signature_def_val.get("inputs") is None:
        raise UserException(
            "unable to find 'inputs' in signature def '{}' for model '{}'".format(
                signature_key, model_name
            )
        )

    parsed_signature = {}
    for input_name, input_metadata in signature_def_val["inputs"].items():
        if input_metadata["tensorShape"] == {}:
            # a scalar with rank 0 and empty shape
            shape = "scalar"
        elif input_metadata["tensorShape"].get("unknownRank", False):
            # unknown rank and shape
            #
            # unknownRank is set to True if the model input has no rank
            # it may lead to an undefined behavior if unknownRank is only checked for its presence
            # so it also gets to be tested against its value
            shape = "unknown"
        elif input_metadata["tensorShape"].get("dim", None):
            # known rank and known/unknown shape
            shape = [int(dim["size"]) for dim in input_metadata["tensorShape"]["dim"]]
        else:
            raise UserException(
                "invalid 'tensorShape' specification for input '{}' in signature key '{}' for model '{}'",
                input_name,
                signature_key,
                model_name,
            )

        parsed_signature[input_name] = {
            "shape": shape if type(shape) == list else [shape],
            "type": DTYPE_TO_TF_TYPE[input_metadata["dtype"]].name,
        }
    return signature_key, parsed_signature


def create_prediction_request(signature_def, signature_key, model_name, model_input):
    prediction_request = predict_pb2.PredictRequest()
    prediction_request.model_spec.name = model_name
    prediction_request.model_spec.signature_name = signature_key

    for column_name, value in model_input.items():
        if signature_def[signature_key]["inputs"][column_name]["tensorShape"] == {}:
            shape = "scalar"
        elif signature_def[signature_key]["inputs"][column_name]["tensorShape"].get(
            "unknownRank", False
        ):
            # unknownRank is set to True if the model input has no rank
            # it may lead to an undefined behavior if unknownRank is only checked for its presence
            # so it also gets to be tested against its value
            shape = "unknown"
        else:
            shape = []
            for dim in signature_def[signature_key]["inputs"][column_name]["tensorShape"]["dim"]:
                shape.append(int(dim["size"]))

        sig_type = signature_def[signature_key]["inputs"][column_name]["dtype"]

        try:
            tensor_proto = tf.compat.v1.make_tensor_proto(value, dtype=DTYPE_TO_TF_TYPE[sig_type])
            prediction_request.inputs[column_name].CopyFrom(tensor_proto)
        except Exception as e:
            if shape == "scalar":
                raise UserException(
                    'key "{}"'.format(column_name), "expected to be a scalar", str(e)
                ) from e
            elif shape == "unknown":
                raise UserException(
                    'key "{}"'.format(column_name), "can be of any rank and shape", str(e)
                ) from e
            else:
                raise UserException(
                    'key "{}"'.format(column_name), "expected shape {}".format(shape), str(e)
                ) from e

    return prediction_request


def parse_response_proto(response_proto):
    results_dict = json_format.MessageToDict(response_proto)
    outputs = results_dict["outputs"]
    outputs_simplified = {}
    for key in outputs:
        value_key = DTYPE_TO_VALUE_KEY[outputs[key]["dtype"]]
        outputs_simplified[key] = outputs[key][value_key]
    return outputs_simplified


def validate_model_input(input_signature, model_input, model_name):
    for input_name, _ in input_signature.items():
        if input_name not in model_input:
            raise UserException("missing key '{}' for model '{}'".format(input_name, model_name))
