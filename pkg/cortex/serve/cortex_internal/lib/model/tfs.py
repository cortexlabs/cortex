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
import time
import grpc
import copy
from typing import Any, Optional, Dict, List, Tuple

from cortex_internal.lib.exceptions import CortexException, UserException
from cortex_internal.lib.log import configure_logger

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


# TensorFlow types
def _define_types() -> Tuple[Dict[str, Any], Dict[str, str]]:
    return (
        {
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
        },
        {
            "DT_INT32": "intVal",
            "DT_INT64": "int64Val",
            "DT_FLOAT": "floatVal",
            "DT_STRING": "stringVal",
            "DT_BOOL": "boolVal",
            "DT_DOUBLE": "doubleVal",
            "DT_HALF": "halfVal",
            "DT_COMPLEX64": "scomplexVal",
            "DT_COMPLEX128": "dcomplexVal",
        },
    )


# for TensorFlowServingAPI
try:
    import tensorflow as tf
    from tensorflow_serving.apis import predict_pb2
    from tensorflow_serving.apis import get_model_metadata_pb2
    from tensorflow_serving.apis import prediction_service_pb2_grpc
    from tensorflow_serving.apis import model_service_pb2_grpc
    from tensorflow_serving.apis import model_management_pb2
    from tensorflow_serving.apis import get_model_status_pb2
    from tensorflow_serving.config import model_server_config_pb2
    from tensorflow_serving.sources.storage_path.file_system_storage_path_source_pb2 import (
        FileSystemStoragePathSourceConfig,
    )

    ServableVersionPolicy = FileSystemStoragePathSourceConfig.ServableVersionPolicy
    Specific = FileSystemStoragePathSourceConfig.ServableVersionPolicy.Specific
    from google.protobuf import json_format

    tensorflow_dependencies_installed = True
    DTYPE_TO_TF_TYPE, DTYPE_TO_VALUE_KEY = _define_types()
    predictRequestClass = predict_pb2.PredictRequest

except ImportError:
    tensorflow_dependencies_installed = False
    predictRequestClass = Any


class TensorFlowServingAPI:
    def __init__(self, address: str):
        """
        TensorFlow Serving API for loading/unloading/reloading TF models and for running predictions.

        Extra arguments passed to the tensorflow/serving container:
            * --max_num_load_retries=0
            * --load_retry_interval_micros=30000000 # 30 seconds
            * --grpc_channel_arguments="grpc.max_concurrent_streams=<processes-per-api-replica>*<threads-per-process>" when inf == 0, otherwise
            * --grpc_channel_arguments="grpc.max_concurrent_streams=<threads-per-process>" when inf > 0.

        Args:
            address: An address with the "host:port" format.
        """

        if not tensorflow_dependencies_installed:
            raise NameError("tensorflow_serving_api and tensorflow packages not installed")

        self.address = address
        self.models = (
            {}
        )  # maps the model ID to the model metadata (signature def, signature key and so on)

        # remove limit for maximum/receive transmission sizes
        options = [
            ("grpc.max_send_message_length", -1),
            ("grpc.max_receive_message_length", -1),
        ]
        self.channel = grpc.insecure_channel(self.address, options=options)

        self._service = model_service_pb2_grpc.ModelServiceStub(self.channel)
        self._pred = prediction_service_pb2_grpc.PredictionServiceStub(self.channel)

    def is_tfs_accessible(self) -> bool:
        """
        Tests whether TFS is accessible or not.
        """
        request = get_model_status_pb2.GetModelStatusRequest()
        request.model_spec.name = "test-model-name"

        try:
            self._service.GetModelStatus(request, timeout=10.0)
        except grpc.RpcError as err:
            if err.code() in [grpc.StatusCode.UNAVAILABLE, grpc.StatusCode.DEADLINE_EXCEEDED]:
                return False
        return True

    def add_single_model(
        self,
        model_name: str,
        model_version: str,
        model_disk_path: str,
        signature_key: Optional[str] = None,
        timeout: Optional[float] = None,
        max_retries: int = 0,
    ) -> None:
        """
        Wrapper for add_models method.
        """
        self.add_models(
            [model_name],
            [[model_version]],
            [model_disk_path],
            [signature_key],
            timeout=timeout,
            max_retries=max_retries,
        )

    def remove_single_model(
        self,
        model_name: str,
        model_version: str,
        timeout: Optional[float] = None,
    ) -> None:
        """
        Wrapper for remove_models method.
        """
        self.remove_models([model_name], [[model_version]], timeout)

    def add_models(
        self,
        model_names: List[str],
        model_versions: List[List[str]],
        model_disk_paths: List[str],
        signature_keys: List[Optional[str]],
        skip_if_present: bool = False,
        timeout: Optional[float] = None,
        max_retries: int = 0,
    ) -> None:
        """
        Add models to TFS. If they can't be loaded, use remove_models to remove them from TFS.

        Args:
            model_names: List of model names to add.
            model_versions: List of lists - each element is a list of versions for a given model name.
            model_disk_paths: The common model disk path of multiple versioned models of the same model name (i.e. modelA/ for modelA/1 and modelA/2).
            skip_if_present: If the models are already loaded, don't make a new request to TFS.
            signature_keys: The signature keys as set in cortex_internal.yaml. If an element is set to None, then "predict" key will be assumed.
            max_retries: How many times to call ReloadConfig before giving up.
        Raises:
            grpc.RpcError in case something bad happens while communicating.
                StatusCode.DEADLINE_EXCEEDED when timeout is encountered. StatusCode.UNAVAILABLE when the service is unreachable.
            cortex_internal.lib.exceptions.CortexException if a non-0 response code is returned (i.e. model couldn't be loaded).
            cortex_internal.lib.exceptions.UserException when a model couldn't be validated for the signature def.
        """

        request = model_management_pb2.ReloadConfigRequest()
        model_server_config = model_server_config_pb2.ModelServerConfig()

        num_added_models = 0
        for model_name, versions, model_disk_path in zip(
            model_names, model_versions, model_disk_paths
        ):
            for model_version in versions:
                versioned_model_disk_path = os.path.join(model_disk_path, model_version)
                num_added_models += self._add_model_to_dict(
                    model_name, model_version, versioned_model_disk_path
                )

        if skip_if_present and num_added_models == 0:
            return

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

        while max_retries >= 0:
            max_retries -= 1
            try:
                # to prevent HandleReloadConfigRequest from
                # throwing an exception (TFS has some race-condition bug)
                time.sleep(0.125)
                response = self._service.HandleReloadConfigRequest(request, timeout)
                break
            except grpc.RpcError as err:
                # to prevent HandleReloadConfigRequest from
                # throwing another exception on the next run
                time.sleep(0.125)
                raise

        if not (response and response.status.error_code == 0):
            if response:
                raise CortexException(
                    "couldn't load user-requested models {} - failed with error code {}: {}".format(
                        model_names, response.status.error_code, response.status.error_message
                    )
                )
            else:
                raise CortexException("couldn't load user-requested models")

        # get models metadata
        for model_name, versions, signature_key in zip(model_names, model_versions, signature_keys):
            for model_version in versions:
                self._load_model_signatures(model_name, model_version, signature_key)

    def remove_models(
        self,
        model_names: List[str],
        model_versions: List[List[str]],
        timeout: Optional[float] = None,
    ) -> None:
        """
        Remove models to TFS.

        Args:
            model_names: List of model names to add.
            model_versions: List of lists - each element is a list of versions for a given model name.
        Raises:
            grpc.RpcError in case something bad happens while communicating.
                StatusCode.DEADLINE_EXCEEDED when timeout is encountered. StatusCode.UNAVAILABLE when the service is unreachable.
            cortex_internal.lib.exceptions.CortexException if a non-0 response code is returned (i.e. model couldn't be unloaded).
        """

        request = model_management_pb2.ReloadConfigRequest()
        model_server_config = model_server_config_pb2.ModelServerConfig()

        for model_name, versions in zip(model_names, model_versions):
            for model_version in versions:
                self._remove_model_from_dict(model_name, model_version)

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

        response = self._service.HandleReloadConfigRequest(request, timeout)

        if not (response and response.status.error_code == 0):
            if response:
                raise CortexException(
                    "couldn't unload user-requested models {} - failed with error code {}: {}".format(
                        model_names, response.status.error_code, response.status.error_message
                    )
                )
            else:
                raise CortexException("couldn't unload user-requested models")

    def poll_available_model_versions(self, model_name: str) -> List[str]:
        """
        Gets the available model versions from TFS.

        Args:
            model_name: The model name to check for versions.

        Returns:
            List of the available versions for the given model from TFS.
        """
        request = get_model_status_pb2.GetModelStatusRequest()
        request.model_spec.name = model_name

        versions = []

        try:
            for model in self._service.GetModelStatus(request).model_version_status:
                if model.state == get_model_status_pb2.ModelVersionStatus.AVAILABLE:
                    versions.append(str(model.version))
        except grpc.RpcError as e:
            pass

        return versions

    def get_registered_model_ids(self) -> List[str]:
        """
        Get the registered model IDs (doesn't poll the TFS server).
        """
        return list(self.models.keys())

    def predict(
        self, model_input: Any, model_name: str, model_version: str, timeout: float = 300.0
    ) -> Any:
        """
        Args:
            model_input: The input to run the prediction on - as passed by the user.
            model_name: Name of the model.
            model_version: Version of the model.
            timeout: How many seconds to wait for the prediction to run before timing out.

        Raises:
            UserException when the model input is not valid or when the model's shape doesn't match that of the input's.
            grpc.RpcError in case something bad happens while communicating - should not happen.

        Returns:
            The prediction.
        """

        model_id = f"{model_name}-{model_version}"

        signature_def = self.models[model_id]["signature_def"]
        signature_key = self.models[model_id]["signature_key"]
        input_signatures = self.models[model_id]["input_signatures"]

        # validate model input
        for input_name, _ in input_signatures.items():
            if input_name not in model_input:
                raise UserException(
                    "missing key '{}' for model '{}' of version '{}'".format(
                        input_name, model_name, model_version
                    )
                )

        # create prediction request
        prediction_request = self._create_prediction_request(
            signature_def, signature_key, model_name, model_version, model_input
        )

        # run prediction
        response_proto = self._pred.Predict(prediction_request, timeout=timeout)

        # interpret response message
        results_dict = json_format.MessageToDict(response_proto)
        outputs = results_dict["outputs"]
        outputs_simplified = {}
        for key in outputs:
            value_key = DTYPE_TO_VALUE_KEY[outputs[key]["dtype"]]
            outputs_simplified[key] = outputs[key][value_key]

        # return parsed response
        return outputs_simplified

    def _remove_model_from_dict(self, model_name: str, model_version: str) -> Tuple[bool, str]:
        model_id = f"{model_name}-{model_version}"
        try:
            model = copy.deepcopy(self.models[model_id])
            del self.models[model_id]
            return True, model
        except KeyError:
            pass
        return False, ""

    def _add_model_to_dict(self, model_name: str, model_version: str, model_disk_path: str) -> bool:
        model_id = f"{model_name}-{model_version}"
        if model_id not in self.models:
            self.models[model_id] = {
                "disk_path": model_disk_path,
            }
            return True
        return False

    def _load_model_signatures(
        self, model_name: str, model_version: str, signature_key: Optional[str] = None
    ) -> None:
        """
        Queries the signature defs from TFS.

        Args:
            model_name: Name of the model.
            model_version: Version of the model.
            signature_key: Signature key of the model as passed in with predictor:signature_key, predictor:models:paths:signature_key or predictor:models:signature_key.
                When set to None, "predict" is the assumed key.

        Raises:
            cortex_internal.lib.exceptions.UserException when the signature def can't be validated.
        """

        # create model metadata request
        request = get_model_metadata_pb2.GetModelMetadataRequest()
        request.model_spec.name = model_name
        request.model_spec.version.value = int(model_version)
        request.metadata_field.append("signature_def")

        # get signature def
        last_idx = 0
        for times in range(100):
            try:
                resp = self._pred.GetModelMetadata(request)
                break
            except grpc.RpcError as e:
                # it has been observed that it may take a little bit of time
                # until a model gets to be accessible with TFS (even though it's already loaded in)
                time.sleep(0.3)
            last_idx = times
        if last_idx == 99:
            raise UserException(
                "couldn't find model '{}' of version '{}' to extract the signature def".format(
                    model_name, model_version
                )
            )

        sigAny = resp.metadata["signature_def"]
        signature_def_map = get_model_metadata_pb2.SignatureDefMap()
        sigAny.Unpack(signature_def_map)
        sigmap = json_format.MessageToDict(signature_def_map)
        signature_def = sigmap["signatureDef"]

        # extract signature key and input signature
        signature_key, input_signatures = self._extract_signatures(
            signature_def, signature_key, model_name, model_version
        )

        model_id = f"{model_name}-{model_version}"
        self.models[model_id]["signature_def"] = signature_def
        self.models[model_id]["signature_key"] = signature_key
        self.models[model_id]["input_signatures"] = input_signatures

    def _get_model_names(self) -> List[str]:
        return list(set([model_id.rsplit("-", maxsplit=1)[0] for model_id in self.models]))

    def _get_model_info(self, model_name: str) -> Tuple[List[str], str]:
        model_disk_path = ""
        versions = []
        for model_id in self.models:
            _model_name, model_version = model_id.rsplit("-", maxsplit=1)
            if _model_name == model_name:
                versions.append(model_version)
                if model_disk_path == "":
                    model_disk_path = os.path.dirname(self.models[model_id]["disk_path"])

        return versions, model_disk_path

    def _extract_signatures(
        self, signature_def, signature_key, model_name: str, model_version: str
    ):
        logger.info(
            "signature defs found in model '{}' for version '{}': {}".format(
                model_name, model_version, signature_def
            )
        )

        available_keys = list(signature_def.keys())
        if len(available_keys) == 0:
            raise UserException(
                "unable to find signature defs in model '{}' of version '{}'".format(
                    model_name, model_version
                )
            )

        if signature_key is None:
            if len(available_keys) == 1:
                logger.info(
                    "signature_key was not configured by user, using signature key '{}' for model '{}' of version '{}' (found in the signature def map)".format(
                        available_keys[0],
                        model_name,
                        model_version,
                    )
                )
                signature_key = available_keys[0]
            elif "predict" in signature_def:
                logger.info(
                    "signature_key was not configured by user, using signature key 'predict' for model '{}' of version '{}' (found in the signature def map)".format(
                        model_name,
                        model_version,
                    )
                )
                signature_key = "predict"
            else:
                raise UserException(
                    "signature_key was not configured by user, please specify one the following keys '{}' for model '{}' of version '{}' (found in the signature def map)".format(
                        ", ".join(available_keys), model_name, model_version
                    )
                )
        else:
            if signature_def.get(signature_key) is None:
                possibilities_str = "key: '{}'".format(available_keys[0])
                if len(available_keys) > 1:
                    possibilities_str = "keys: '{}'".format("', '".join(available_keys))

                raise UserException(
                    "signature_key '{}' was not found in signature def map for model '{}' of version '{}', but found the following {}".format(
                        signature_key, model_name, model_version, possibilities_str
                    )
                )

        signature_def_val = signature_def.get(signature_key)

        if signature_def_val.get("inputs") is None:
            raise UserException(
                "unable to find 'inputs' in signature def '{}' for model '{}'".format(
                    signature_key, model_name
                )
            )

        parsed_signatures = {}
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

            parsed_signatures[input_name] = {
                "shape": shape if type(shape) == list else [shape],
                "type": DTYPE_TO_TF_TYPE[input_metadata["dtype"]].name,
            }
        return signature_key, parsed_signatures

    def _create_prediction_request(
        self,
        signature_def: dict,
        signature_key: str,
        model_name: str,
        model_version: int,
        model_input: Any,
    ) -> predictRequestClass:
        prediction_request = predict_pb2.PredictRequest()
        prediction_request.model_spec.name = model_name
        prediction_request.model_spec.version.value = int(model_version)
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
                for dim in signature_def[signature_key]["inputs"][column_name]["tensorShape"][
                    "dim"
                ]:
                    shape.append(int(dim["size"]))

            sig_type = signature_def[signature_key]["inputs"][column_name]["dtype"]

            try:
                tensor_proto = tf.compat.v1.make_tensor_proto(
                    value, dtype=DTYPE_TO_TF_TYPE[sig_type]
                )
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


class TensorFlowServingAPIClones:
    """
    TFS API to load/unload models from multiple TFS server clones. Built on top of TensorFlowServingAPI.
    """

    def __init__(self, addresses: List[str]):
        """
        Args:
            addresses: A list of addresses with the "host:port" format.
        """

        if len(addresses) == 0:
            raise ValueError("addresses list must have at least one address")
        self._clients = [TensorFlowServingAPI(address) for address in addresses]

    def is_tfs_accessible(self) -> bool:
        """
        Tests whether all TFS servers are accessible or not.
        """
        return all([client.is_tfs_accessible() for client in self._clients])

    def add_single_model(
        self,
        model_name: str,
        model_version: str,
        model_disk_path: str,
        signature_key: Optional[str] = None,
        timeout: Optional[float] = None,
        max_retries: int = 0,
    ) -> None:
        """
        Wrapper for add_models method.
        """
        for client in self._clients:
            client.add_single_model(
                model_name, model_version, model_disk_path, signature_key, timeout, max_retries
            )

    def remove_single_model(
        self,
        model_name: str,
        model_version: str,
        timeout: Optional[float] = None,
    ) -> None:
        """
        Wrapper for remove_models method.
        """
        for client in self._clients:
            client.remove_single_model(model_name, model_version, timeout)

    def add_models(
        self,
        model_names: List[str],
        model_versions: List[List[str]],
        model_disk_paths: List[str],
        signature_keys: List[Optional[str]],
        skip_if_present: bool = False,
        timeout: Optional[float] = None,
        max_retries: int = 0,
    ) -> None:
        """
        Add the same models to multiple TFS servers. If they can't be loaded, use remove_models to remove them from TFS.

        Args:
            model_names: List of model names to add.
            model_versions: List of lists - each element is a list of versions for a given model name.
            model_disk_paths: The common model disk path of multiple versioned models of the same model name (i.e. modelA/ for modelA/1 and modelA/2).
            skip_if_present: If the models are already loaded, don't make a new request to TFS.
            signature_keys: The signature keys as set in cortex_internal.yaml. If an element is set to None, then "predict" key will be assumed.
            max_retries: How many times to call ReloadConfig before giving up.
        Raises:
            grpc.RpcError in case something bad happens while communicating.
                StatusCode.DEADLINE_EXCEEDED when timeout is encountered. StatusCode.UNAVAILABLE when the service is unreachable.
            cortex_internal.lib.exceptions.CortexException if a non-0 response code is returned (i.e. model couldn't be loaded).
            cortex_internal.lib.exceptions.UserException when a model couldn't be validated for the signature def.
        """
        for client in self._clients:
            client.add_models(
                model_names,
                model_versions,
                model_disk_paths,
                signature_keys,
                skip_if_present,
                timeout,
                max_retries,
            )

    def remove_models(
        self,
        model_names: List[str],
        model_versions: List[List[str]],
        timeout: Optional[float] = None,
    ) -> None:
        """
        Remove the same models from multiple TFS servers.

        Args:
            model_names: List of model names to add.
            model_versions: List of lists - each element is a list of versions for a given model name.
        Raises:
            grpc.RpcError in case something bad happens while communicating.
                StatusCode.DEADLINE_EXCEEDED when timeout is encountered. StatusCode.UNAVAILABLE when the service is unreachable.
            cortex_internal.lib.exceptions.CortexException if a non-0 response code is returned (i.e. model couldn't be unloaded).
        """
        for client in self._clients:
            client.remove_models(model_names, model_versions, timeout)

    def poll_available_model_versions(self, model_name: str) -> List[str]:
        """
        Gets the available model versions from TFS.
        Since all TFS servers are assumed to have the same models in memory, it makes sense to just poll one.

        Args:
            model_name: The model name to check for versions.

        Returns:
            List of the available versions for the given model from TFS.
        """

        return self._clients[0].poll_available_model_versions(model_name)

    def get_registered_model_ids(self) -> List[str]:
        """
        Get the registered model IDs (doesn't poll the TFS server).
        Since all TFS servers are assumed to have the same models in memory, it makes sense to just poll one.
        """
        return self._clients[0].get_registered_model_ids()

    @property
    def models(self) -> dict:
        return self._clients[0].models
