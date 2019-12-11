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


def create_get_model_metadata_request():
    get_model_metadata_request = get_model_metadata_pb2.GetModelMetadataRequest()
    get_model_metadata_request.model_spec.name = "model"
    get_model_metadata_request.metadata_field.append("signature_def")
    return get_model_metadata_request


def run_get_model_metadata():
    request = create_get_model_metadata_request()
    resp = local_cache["stub"].GetModelMetadata(request, timeout=10.0)
    sigAny = resp.metadata["signature_def"]
    signature_def_map = get_model_metadata_pb2.SignatureDefMap()
    sigAny.Unpack(signature_def_map)
    sigmap = json_format.MessageToDict(signature_def_map)
    return sigmap


def extract_signature(signature_def, signature_key):
    cx_logger().info("signature defs found in model: {}".format(signature_def))

    available_keys = list(signature_def.keys())
    if len(available_keys) == 0:
        raise UserException("unable to find signature defs in model")

    if signature_key is None:
        if len(available_keys) == 1:
            cx_logger().info(
                "signature_key was not configured by user, using signature key '{}' (found in the signature def map)".format(
                    available_keys[0]
                )
            )
            signature_key = available_keys[0]
        elif "predict" in signature_def:
            cx_logger().info(
                "signature_key was not configured by user, using signature key 'predict' (found in the signature def map)"
            )
            signature_key = "predict"
        else:
            raise UserException(
                "signature_key was not configured by user, please specify one the following keys '{}' (found in the signature def map)".format(
                    "', '".join(available_keys)
                )
            )
    else:
        if signature_def.get(signature_key) is None:
            possibilities_str = "key: '{}'".format(available_keys[0])
            if len(available_keys) > 1:
                possibilities_str = "keys: '{}'".format("', '".join(available_keys))

            raise UserException(
                "signature_key '{}' was not found in signature def map, but found the following {}".format(
                    signature_key, possibilities_str
                )
            )

    signature_def_val = signature_def.get(signature_key)

    if signature_def_val.get("inputs") is None:
        raise UserException("unable to find 'inputs' in signature def '{}'".format(signature_key))

    parsed_signature = {}
    for input_name, input_metadata in signature_def_val["inputs"].items():
        parsed_signature[input_name] = {
            "shape": [int(dim["size"]) for dim in input_metadata["tensorShape"]["dim"]],
            "type": DTYPE_TO_TF_TYPE[input_metadata["dtype"]].name,
        }
    return signature_key, parsed_signature


class TFClient:
    def __init__(self, tf_serving_url, signature_key):
        self.tf_serving_url = tf_serving_url
        channel = grpc.insecure_channel(tf_serving_url)
        self.stub = prediction_service_pb2_grpc.PredictionServiceStub(channel)
        limit = 60
        for i in range(limit):
            try:
                self.signature = run_get_model_metadata()["signatureDef"]
                break
            except Exception as e:
                if i > 6:
                    cx_logger().warn(
                        "unable to read model metadata - model is still loading, retrying..."
                    )
                if i == limit - 1:
                    cx_logger().exception("retry limit exceeded")
                    sys.exit(1)

            time.sleep(5)

        parsed_signature_key, parsed_signature = extract_signature(self.signature, signature_key)
        self.signature_key = parsed_signature_key
        self.parsed_signature = parsed_signature

    def prediction_service_stub(self):
        return self.stub

    def _create_prediction_request(self, payload):
        signature_def = self.signature
        signature_key = self.signature_key
        prediction_request = predict_pb2.PredictRequest()
        prediction_request.model_spec.name = "model"
        prediction_request.model_spec.signature_name = signature_key

        for column_name, value in payload.items():
            shape = []
            for dim in signature_def[signature_key]["inputs"][column_name]["tensorShape"]["dim"]:
                shape.append(int(dim["size"]))

            sig_type = signature_def[signature_key]["inputs"][column_name]["dtype"]

            try:
                tensor_proto = tf.compat.v1.make_tensor_proto(
                    value, dtype=DTYPE_TO_TF_TYPE[sig_type]
                )
                prediction_request.inputs[column_name].CopyFrom(tensor_proto)
            except Exception as e:
                raise UserException(
                    'key "{}"'.format(column_name), "expected shape {}".format(shape), str(e)
                ) from e

        return prediction_request

    def _parse_response_proto(self, response_proto):
        results_dict = json_format.MessageToDict(response_proto)
        outputs = results_dict["outputs"]
        outputs_simplified = {}
        for key in outputs:
            value_key = DTYPE_TO_VALUE_KEY[outputs[key]["dtype"]]
            outputs_simplified[key] = outputs[key][value_key]
        return outputs_simplified

    def _validate_payload(self, payload):
        signature = self.parsed_signature
        for input_name, _ in signature.items():
            if input_name not in payload:
                raise UserException('missing key "{}"'.format(input_name))

    def signature(self):
        return self.metadata["signatureDef"]

    def predict(self, payload):
        self._validate_payload(payload)
        prediction_request = self._create_prediction_request(payload)
        response_proto = self.stub.Predict(prediction_request, timeout=300.0)
        return self._parse_response_proto(response_proto)

