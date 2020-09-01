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
import time
import sys
import grpc
import threading as td
import copy
from typing import Any, Optional, Callable, List, Tuple

# for TensorFlowClient
import tensorflow as tf
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import get_model_metadata_pb2
from tensorflow_serving.apis import prediction_service_pb2_grpc
from google.protobuf import json_format

# for TensorFlowServingAPI
from tensorflow_serving.apis import model_service_pb2_grpc
from tensorflow_serving.apis import model_management_pb2
from tensorflow_serving.config import model_server_config_pb2
from tensorflow_serving.sources.storage_path.file_system_storage_path_source_pb2 import (
    FileSystemStoragePathSourceConfig,
)

ServableVersionPolicy = FileSystemStoragePathSourceConfig.ServableVersionPolicy
Specific = FileSystemStoragePathSourceConfig.ServableVersionPolicy.Specific

from cortex.lib.exceptions import UserRuntimeException, CortexException, UserException, WithBreak
from cortex.lib.concurrency import LockedFile

# from cortex.lib.model import (
#     ModelsHolder,
#     LockedModel,
#     ModelsTree,
#     LockedModelsTree,
#     CuratedModelResources,
#     find_ondisk_model_info,
#     find_ondisk_models,
# )
from cortex.lib.log import cx_logger
from cortex import consts

logger = cx_logger()


# class TensorFlowClient:
#     def __init__(self, tf_serving_url, models=None):
#         """Setup gRPC connection to TensorFlow Serving container.

#         Args:
#             tf_serving_url (string): Localhost URL to TF Serving container.
#             models        ([Model]): List of models deployed with TF serving container.
#         """
#         self._tf_serving_url = tf_serving_url
#         self._models = models
#         self._model_names = get_model_names(models)

#         channel = grpc.insecure_channel(tf_serving_url)
#         self._stub = prediction_service_pb2_grpc.PredictionServiceStub(channel)

#         self._signatures = get_signature_defs(self._stub, models)
#         parsed_signature_keys, parsed_signatures = extract_signatures(
#             self._signatures, get_model_signature_map(models)
#         )
#         self._signature_keys = parsed_signature_keys
#         self._input_signatures = parsed_signatures

#     def predict(self, model_input, model_name=None):
#         """Validate model_input, convert it to a Prediction Proto, and make a request to TensorFlow Serving.

#         Args:
#             model_input: Input to the model.
#             model_name: Model to use when multiple models are deployed in a single API.

#         Returns:
#             dict: TensorFlow Serving response converted to a dictionary.
#         """
#         if consts.SINGLE_MODEL_NAME in self._model_names:
#             return self._run_inference(model_input, consts.SINGLE_MODEL_NAME)

#         if model_name is None:
#             raise UserRuntimeException(
#                 "model_name was not specified, choose one of the following: {}".format(
#                     self._model_names
#                 )
#             )

#         if model_name not in self._model_names:
#             raise UserRuntimeException(
#                 "'{}' model wasn't found in the list of available models: {}".format(
#                     model_name, self._model_names
#                 )
#             )

#         return self._run_inference(model_input, model_name)

#     def _run_inference(self, model_input, model_name):
#         input_signature = self._input_signatures[model_name]
#         signature = self._signatures[model_name]
#         signature_key = self._signature_keys[model_name]

#         validate_model_input(input_signature, model_input, model_name)
#         prediction_request = create_prediction_request(
#             signature, signature_key, model_name, model_input
#         )
#         response_proto = self._stub.Predict(prediction_request, timeout=300.0)
#         return parse_response_proto(response_proto)

#     @property
#     def stub(self):
#         return self._stub

#     @property
#     def input_signatures(self):
#         return self._input_signatures


# DTYPE_TO_TF_TYPE = {
#     "DT_FLOAT": tf.float32,
#     "DT_DOUBLE": tf.float64,
#     "DT_INT32": tf.int32,
#     "DT_UINT8": tf.uint8,
#     "DT_INT16": tf.int16,
#     "DT_INT8": tf.int8,
#     "DT_STRING": tf.string,
#     "DT_COMPLEX64": tf.complex64,
#     "DT_INT64": tf.int64,
#     "DT_BOOL": tf.bool,
#     "DT_QINT8": tf.qint8,
#     "DT_QUINT8": tf.quint8,
#     "DT_QINT32": tf.qint32,
#     "DT_BFLOAT16": tf.bfloat16,
#     "DT_QINT16": tf.qint16,
#     "DT_QUINT16": tf.quint16,
#     "DT_UINT16": tf.uint16,
#     "DT_COMPLEX128": tf.complex128,
#     "DT_HALF": tf.float16,
#     "DT_RESOURCE": tf.resource,
#     "DT_VARIANT": tf.variant,
#     "DT_UINT32": tf.uint32,
#     "DT_UINT64": tf.uint64,
# }

# DTYPE_TO_VALUE_KEY = {
#     "DT_INT32": "intVal",
#     "DT_INT64": "int64Val",
#     "DT_FLOAT": "floatVal",
#     "DT_STRING": "stringVal",
#     "DT_BOOL": "boolVal",
#     "DT_DOUBLE": "doubleVal",
#     "DT_HALF": "halfVal",
#     "DT_COMPLEX64": "scomplexVal",
#     "DT_COMPLEX128": "dcomplexVal",
# }


# def get_signature_defs(stub, models):
#     sigmaps = {}
#     for model in models:
#         sigmaps[model.name] = get_signature_def(stub, model)

#     return sigmaps


# def get_signature_def(stub, model):
#     limit = 2
#     for i in range(limit):
#         try:
#             request = create_get_model_metadata_request(model.name)
#             resp = stub.GetModelMetadata(request, timeout=10.0)
#             sigAny = resp.metadata["signature_def"]
#             signature_def_map = get_model_metadata_pb2.SignatureDefMap()
#             sigAny.Unpack(signature_def_map)
#             sigmap = json_format.MessageToDict(signature_def_map)
#             return sigmap["signatureDef"]
#         except Exception as e:
#             print(e)
#             cx_logger().warn(
#                 "unable to read model metadata for model '{}' - retrying ...".format(model.name)
#             )

#         time.sleep(5)

#     raise CortexException(
#         "timeout: unable to read model metadata for model '{}'".format(model.name)
#     )


# def create_get_model_metadata_request(model_name):
#     get_model_metadata_request = get_model_metadata_pb2.GetModelMetadataRequest()
#     get_model_metadata_request.model_spec.name = model_name
#     get_model_metadata_request.metadata_field.append("signature_def")
#     return get_model_metadata_request


# def extract_signatures(signature_defs, signature_keys):
#     parsed_signature_keys = {}
#     parsed_signatures = {}
#     for model_name in signature_defs:
#         parsed_signature_key, parsed_signature = extract_signature(
#             signature_defs[model_name],
#             signature_keys[model_name],
#             model_name,
#         )
#         parsed_signature_keys[model_name] = parsed_signature_key
#         parsed_signatures[model_name] = parsed_signature

#     return parsed_signature_keys, parsed_signatures


# def extract_signature(signature_def, signature_key, model_name):
#     cx_logger().info("signature defs found in model '{}': {}".format(model_name, signature_def))

#     available_keys = list(signature_def.keys())
#     if len(available_keys) == 0:
#         raise UserException("unable to find signature defs in model '{}'".format(model_name))

#     if signature_key is None:
#         if len(available_keys) == 1:
#             cx_logger().info(
#                 "signature_key was not configured by user, using signature key '{}' for model '{}' (found in the signature def map)".format(
#                     available_keys[0],
#                     model_name,
#                 )
#             )
#             signature_key = available_keys[0]
#         elif "predict" in signature_def:
#             cx_logger().info(
#                 "signature_key was not configured by user, using signature key 'predict' for model '{}' (found in the signature def map)".format(
#                     model_name
#                 )
#             )
#             signature_key = "predict"
#         else:
#             raise UserException(
#                 "signature_key was not configured by user, please specify one the following keys '{}' for model '{}' (found in the signature def map)".format(
#                     ", ".join(available_keys), model_name
#                 )
#             )
#     else:
#         if signature_def.get(signature_key) is None:
#             possibilities_str = "key: '{}'".format(available_keys[0])
#             if len(available_keys) > 1:
#                 possibilities_str = "keys: '{}'".format("', '".join(available_keys))

#             raise UserException(
#                 "signature_key '{}' was not found in signature def map for model '{}', but found the following {}".format(
#                     signature_key, model_name, possibilities_str
#                 )
#             )

#     signature_def_val = signature_def.get(signature_key)

#     if signature_def_val.get("inputs") is None:
#         raise UserException(
#             "unable to find 'inputs' in signature def '{}' for model '{}'".format(
#                 signature_key, model_name
#             )
#         )

#     parsed_signature = {}
#     for input_name, input_metadata in signature_def_val["inputs"].items():
#         parsed_signature[input_name] = {
#             "shape": [int(dim["size"]) for dim in input_metadata["tensorShape"]["dim"]],
#             "type": DTYPE_TO_TF_TYPE[input_metadata["dtype"]].name,
#         }
#     return signature_key, parsed_signature


# def create_prediction_request(signature_def, signature_key, model_name, model_input):
#     prediction_request = predict_pb2.PredictRequest()
#     prediction_request.model_spec.name = model_name
#     prediction_request.model_spec.signature_name = signature_key

#     for column_name, value in model_input.items():
#         shape = []
#         for dim in signature_def[signature_key]["inputs"][column_name]["tensorShape"]["dim"]:
#             shape.append(int(dim["size"]))

#         sig_type = signature_def[signature_key]["inputs"][column_name]["dtype"]

#         try:
#             tensor_proto = tf.compat.v1.make_tensor_proto(value, dtype=DTYPE_TO_TF_TYPE[sig_type])
#             prediction_request.inputs[column_name].CopyFrom(tensor_proto)
#         except Exception as e:
#             raise UserException(
#                 'key "{}"'.format(column_name), "expected shape {}".format(shape), str(e)
#             ) from e

#     return prediction_request


# def parse_response_proto(response_proto):
#     results_dict = json_format.MessageToDict(response_proto)
#     outputs = results_dict["outputs"]
#     outputs_simplified = {}
#     for key in outputs:
#         value_key = DTYPE_TO_VALUE_KEY[outputs[key]["dtype"]]
#         outputs_simplified[key] = outputs[key][value_key]
#     return outputs_simplified


# def validate_model_input(input_signature, model_input, model_name):
#     for input_name, _ in input_signature.items():
#         if input_name not in model_input:
#             raise UserException("missing key '{}' for model '{}'".format(input_name, model_name))


class TensorFlowServingAPI:
    def __init__(self, address: str):
        """
        TensorFlow Serving API for loading/unloading/reloading TF models.

        TODO extra arguments to pass to the tensorflow/serving container:
            * --max_num_load_retries=0
            * --load_retry_interval_micros=30000000 # 30 seconds
            * --grpc_channel_arguments="grpc.max_concurrent_streams=<processes-per-api-replica>*<threads-per-process>"

        Args:
            address: An address with the "host:port" format.
        """

        self.address = address
        self.models = {}

        self.channel = grpc.insecure_channel(self.address)
        self.stub = model_service_pb2_grpc.ModelServiceStub(self.channel)

    def add_single_model(
        self,
        model_name: str,
        model_version: str,
        model_disk_path: str,
        timeout: Optional[float] = None,
    ) -> None:
        """
        Wrapper for add_models method.
        """
        self.add_models([model_name], [[model_version]], [model_disk_path], timeout)

    def remove_single_model(
        self,
        model_name: str,
        model_version: str,
        model_disk_path: str,
        timeout: Optional[float] = None,
    ) -> None:
        """
        Wrapper for remove_models method.
        """
        self.remove_models([model_name], [[model_version]], [model_disk_path], timeout)

    def add_models(
        self,
        model_names: List[str],
        model_versions: List[List[str]],
        model_disk_paths: List[str],
        timeout: Optional[float] = None,
    ) -> None:
        """
        Add models to TFS.

        Args:
            model_names: List of model names to add.
            model_versions: List of lists - each element is a list of versions for a given model name.
            model_disk_paths: The common model disk path of multiple versioned models of the same model name (i.e. modelA/ for modelA/1 and modelA/2).
        Raises:
            grpc.RpcError in case something bad happens while communicating.
                StatusCode.DEADLINE_EXCEEDED when timeout is encountered. StatusCode.UNAVAILABLE when the service is unreachable.
            cortex.lib.exceptions.CortexException if a non-0 response code is returned (i.e. model couldn't be loaded).
        """

        request = model_management_pb2.ReloadConfigRequest()
        model_server_config = model_server_config_pb2.ModelServerConfig()

        added_models = []
        for model_name, versions, model_disk_path in zip(
            model_names, model_versions, model_disk_paths
        ):
            for model_version in versions:
                versioned_model_disk_path = os.path.join(model_disk_path, model_version)
                success = self._add_model_to_dict(
                    model_name, model_version, versioned_model_disk_path
                )
                if success:
                    added_models.append((model_name, model_version))

        config_list = model_server_config_pb2.ModelConfigList()
        current_model_names = self._get_model_names()
        for model_name in current_model_names:
            versions, model_disk_path = self._get_model_info(model_name)
            versions = [int(version) for version in versions]
            model_config = config_list.config.add()
            model_config.name = model_name
            model_config.base_path = model_disk_path
            model_config.model_version_policy.CopyFrom(
                ServableVersionPolicy(specific=Specific(versions=versions))
            )
            model_config.model_platform = "tensorflow"

        model_server_config.model_config_list.CopyFrom(config_list)
        request.config.CopyFrom(model_server_config)

        try:
            response = self.stub.HandleReloadConfigRequest(request, timeout)
            error = None
        except grpc.RpcError as e:
            error = e

        if error or not (response and response.status.error_code == 0):
            for model_name, model_version in added_models:
                self._remove_model_from_dict(model_name, model_version)

            if error:
                raise error

            if response:
                raise CortexException(
                    "couldn't unload user-requested models - failed with error code {}: {}".format(
                        response.status.error_code, response.status.error_message
                    )
                )
            else:
                raise CortexException("couldn't unload user-requested models")

        if not (response and response.status.error_code == 0):
            if response:
                raise CortexException(
                    "couldn't load user-requested models - failed with error code {}: {}".format(
                        response.status.error_code, response.status.error_message
                    )
                )
            else:
                raise CortexException("couldn't load user-requested models")

    def remove_models(
        self,
        model_names: List[str],
        model_versions: List[List[str]],
        model_disk_paths: List[str],
        timeout: Optional[float] = None,
    ) -> None:
        """
        Add models to TFS.

        Args:
            model_names: List of model names to add.
            model_versions: List of lists - each element is a list of versions for a given model name.
            model_disk_paths: The common model disk path of multiple versioned models of the same model name (i.e. modelA/ for modelA/1 and modelA/2).
        Raises:
            grpc.RpcError in case something bad happens while communicating.
                StatusCode.DEADLINE_EXCEEDED when timeout is encountered. StatusCode.UNAVAILABLE when the service is unreachable.
            cortex.lib.exceptions.CortexException if a non-0 response code is returned (i.e. model couldn't be unloaded).
        """

        request = model_management_pb2.ReloadConfigRequest()
        model_server_config = model_server_config_pb2.ModelServerConfig()

        removed_models = []
        for model_name, versions, model_disk_path in zip(
            model_names, model_versions, model_disk_paths
        ):
            for model_version in versions:
                success, disk_path = self._remove_model_from_dict(model_name, model_version)
                if not success:
                    continue
                removed_models.append((model_name, model_version, disk_path))

        config_list = model_server_config_pb2.ModelConfigList()
        remaining_model_names = self._get_model_names()
        for model_name in remaining_model_names:
            versions, model_disk_path = self._get_model_info(model_name)
            versions = [int(version) for version in versions]
            model_config = config_list.config.add()
            model_config.name = model_name
            model_config.base_path = model_disk_path
            model_config.model_version_policy.CopyFrom(
                ServableVersionPolicy(specific=Specific(versions=versions))
            )
            model_config.model_platform = "tensorflow"

        model_server_config.model_config_list.CopyFrom(config_list)
        request.config.CopyFrom(model_server_config)

        try:
            response = self.stub.HandleReloadConfigRequest(request, timeout)
            error = None
        except grpc.RpcError as e:
            error = e

        if error or not (response and response.status.error_code == 0):
            for model_name, model_version, model_disk_path in removed_models:
                self._add_model_to_dict(model_name, model_version, model_disk_path)

            if error:
                raise error

            if response:
                raise CortexException(
                    "couldn't unload user-requested models - failed with error code {}: {}".format(
                        response.status.error_code, response.status.error_message
                    )
                )
            else:
                raise CortexException("couldn't unload user-requested models")

    def refresh(self, timeout: Optional[float] = None) -> None:
        """
        Reloads existing models if they have changed on disk.
        Removes models from TFS that failed to load.

        Note: doesn't appear to be reloading models that have changed on disk. Probably the best way is to
        remove_single_model and then call add_single_model to reload a versioned model.

        Raises:
            grpc.RpcError in case something bad happens while communicating.
                StatusCode.DEADLINE_EXCEEDED when timeout is encountered. StatusCode.UNAVAILABLE when the service is unreachable.
            cortex.lib.exceptions.CortexException if a non-0 response code is returned (i.e. model couldn't be reloaded).
        """

        request = model_management_pb2.ReloadConfigRequest()
        model_server_config = model_server_config_pb2.ModelServerConfig()
        config_list = model_server_config_pb2.ModelConfigList()

        remaining_model_names = self._get_model_names()
        for model_name in remaining_model_names:
            versions, model_disk_path = self._get_model_info(model_name)
            versions = [int(version) for version in versions]
            model_config = config_list.config.add()
            model_config.name = model_name
            model_config.base_path = model_disk_path
            model_config.model_version_policy.CopyFrom(
                ServableVersionPolicy(specific=Specific(versions=versions))
            )
            model_config.model_platform = "tensorflow"

        model_server_config.model_config_list.CopyFrom(config_list)
        request.config.CopyFrom(model_server_config)

        response = self.stub.HandleReloadConfigRequest(request, timeout)

        if not (response and response.status.error_code == 0):
            if response:
                raise CortexException(
                    "couldn't reload user-requested models - failed with error code {}: {}".format(
                        response.status.error_code, response.status.error_message
                    )
                )
            else:
                raise CortexException("couldn't reload user-requested models")

    def _remove_model_from_dict(self, model_name: str, model_version: str) -> Tuple[bool, str]:
        model_id = f"{model_name}-{model_version}"
        try:
            disk_path = copy.deepcopy(self.models[model_id])
            del self.models[model_id]
            return True, disk_path
        except KeyError:
            pass
        return False, ""

    def _add_model_to_dict(self, model_name: str, model_version: str, model_disk_path: str) -> bool:
        model_id = f"{model_name}-{model_version}"
        if model_id not in self.models:
            self.models[model_id] = model_disk_path
            return True
        return False

    def _get_model_names(self) -> List[str]:
        return [model_id.rsplit("-")[0] for model_id in self.models]

    def _get_model_info(self, model_name: str) -> Tuple[List[str], str]:
        model_disk_path = ""
        versions = []
        for model_id in self.models:
            _model_name, model_version = model_id.rsplit("-")
            if _model_name == model_name:
                versions.append(model_version)
                if model_disk_path == "":
                    model_disk_path = os.path.dirname(self.models[model_id])

        return versions, model_disk_path
