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
import copy
import grpc
import threading as td
from typing import Any, Dict, Optional, List

from cortex_internal.lib.exceptions import (
    UserRuntimeException,
    UserException,
    WithBreak,
)
from cortex_internal.lib.model import (
    TensorFlowServingAPI,
    ModelsHolder,
    ModelsTree,
    LockedModel,
    LockedModelsTree,
    get_models_from_api_spec,
)
from cortex_internal import consts
from cortex_internal.lib.log import configure_logger

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


class TensorFlowClient:
    def __init__(
        self,
        tf_serving_url,
        api_spec: dict,
        models: Optional[ModelsHolder],
        model_dir: Optional[str],
        models_tree: Optional[ModelsTree],
    ):
        """
        Setup gRPC connection to TensorFlow Serving container.

        Args:
            tf_serving_url: Localhost URL to TF Serving container (i.e. "localhost:9000")
            api_spec: API configuration.

            models: Holding all models into memory. Only when processes_per_replica = 1 and caching enabled.
            model_dir: Where the models are saved on disk. Only when processes_per_replica = 1 and caching enabled.
            models_tree: A tree of the available models from upstream. Only when processes_per_replica = 1 and caching enabled.
        """

        self.tf_serving_url = tf_serving_url

        self._api_spec = api_spec
        self._models = models
        self._models_tree = models_tree
        self._model_dir = model_dir

        self._spec_models = get_models_from_api_spec(api_spec)

        if (
            self._api_spec["predictor"]["models"]
            and self._api_spec["predictor"]["models"]["dir"] is not None
        ):
            self._models_dir = True
        else:
            self._models_dir = False
            self._spec_model_names = self._spec_models.get_field("name")

        self._multiple_processes = self._api_spec["predictor"]["processes_per_replica"] > 1
        self._caching_enabled = self._is_model_caching_enabled()

        if self._models:
            self._models.set_callback("load", self._load_model)
            self._models.set_callback("remove", self._remove_models)

        self._client = TensorFlowServingAPI(tf_serving_url)

    def predict(
        self, model_input: Any, model_name: Optional[str] = None, model_version: str = "latest"
    ) -> dict:
        """
        Validate model_input, convert it to a Prediction Proto, and make a request to TensorFlow Serving.

        Args:
            model_input: Input to the model.
            model_name (optional): Name of the model to retrieve (when multiple models are deployed in an API).
                When predictor.models.paths is specified, model_name should be the name of one of the models listed in the API config.
                When predictor.models.dir is specified, model_name should be the name of a top-level directory in the models dir.
            model_version (string, optional): Version of the model to retrieve. Can be omitted or set to "latest" to select the highest version.

        Returns:
            dict: TensorFlow Serving response converted to a dictionary.
        """

        if model_version != "latest" and not model_version.isnumeric():
            raise UserRuntimeException(
                "model_version must be either a parse-able numeric value or 'latest'"
            )

        # when ppredictor:models:path or predictor:models:paths is specified
        if not self._models_dir:

            # when predictor:models:path is provided
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
        if self._models_dir and model_name is None:
            raise UserRuntimeException("model_name was not specified")

        return self._run_inference(model_input, model_name, model_version)

    def _run_inference(self, model_input: Any, model_name: str, model_version: str) -> dict:
        """
        When processes_per_replica = 1 and caching enabled, check/load model and make prediction.
        When processes_per_replica > 0 and caching disabled, attempt to make prediction regardless.

        Args:
            model_input: Input to the model.
            model_name: Name of the model, as it's specified in predictor:models:paths or in the other case as they are named on disk.
            model_version: Version of the model, as it's found on disk. Can also infer the version number from the "latest" version tag.

        Returns:
            The prediction.
        """

        model = None
        tag = ""
        if model_version == "latest":
            tag = model_version

        if not self._caching_enabled:

            # determine model version
            if tag == "latest":
                versions = self._client.poll_available_model_versions(model_name)
                if len(versions) == 0:
                    raise UserException(
                        f"model '{model_name}' accessed with tag {tag} couldn't be found"
                    )
                model_version = str(max(map(lambda x: int(x), versions)))
            model_id = model_name + "-" + model_version

            return self._client.predict(model_input, model_name, model_version)

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
            logger.info(f"grabbing access to model {model_name} of version {model_version}")
            with LockedModelsTree(self._models_tree, "r", model_name, model_version):

                # check if the versioned model exists
                model_id = model_name + "-" + model_version
                if model_id not in self._models_tree:
                    available_model = False
                    logger.info(f"model {model_name} of version {model_version} is not available")
                    raise WithBreak

                # retrieve model tree's metadata
                upstream_model = self._models_tree[model_id]
                current_upstream_ts = int(upstream_model["timestamp"].timestamp())
                logger.info(f"model {model_name} of version {model_version} is available")

            if not available_model:
                if tag == "":
                    raise UserException(
                        f"model '{model_name}' of version '{model_version}' couldn't be found"
                    )
                raise UserException(
                    f"model '{model_name}' accessed with tag '{tag}' couldn't be found"
                )

            # grab shared access to models holder and retrieve model
            update_model = False
            prediction = None
            tfs_was_unresponsive = False
            with LockedModel(self._models, "r", model_name, model_version):
                logger.info(f"checking the {model_name} {model_version} status")
                status, local_ts = self._models.has_model(model_name, model_version)
                if status in ["not-available", "on-disk"] or (
                    status != "not-available" and local_ts != current_upstream_ts
                ):
                    logger.info(
                        f"model {model_name} of version {model_version} is not loaded (with status {status} or different timestamp)"
                    )
                    update_model = True
                    raise WithBreak

                # run prediction
                logger.info(f"run the prediction on model {model_name} of version {model_version}")
                self._models.get_model(model_name, model_version, tag)
                try:
                    prediction = self._client.predict(model_input, model_name, model_version)
                except grpc.RpcError as e:
                    # effectively when it got restarted
                    if len(self._client.poll_available_model_versions(model_name)) > 0:
                        raise
                    tfs_was_unresponsive = True

            # remove model from disk and memory references if TFS gets unresponsive
            if tfs_was_unresponsive:
                with LockedModel(self._models, "w", model_name, model_version):
                    available_versions = self._client.poll_available_model_versions(model_name)
                    status, _ = self._models.has_model(model_name, model_version)
                    if not (status == "in-memory" and model_version not in available_versions):
                        raise WithBreak

                    logger.info(
                        f"removing model {model_name} of version {model_version} because TFS got unresponsive"
                    )
                    self._models.remove_model(model_name, model_version)

            # download, load into memory the model and retrieve it
            if update_model:
                # grab exclusive access to models holder
                with LockedModel(self._models, "w", model_name, model_version):

                    # check model status
                    status, local_ts = self._models.has_model(model_name, model_version)

                    # refresh disk model
                    if status == "not-available" or (
                        status in ["on-disk", "in-memory"] and local_ts != current_upstream_ts
                    ):
                        # unload model from TFS
                        if status == "in-memory":
                            try:
                                logger.info(
                                    f"unloading model {model_name} of version {model_version} from TFS"
                                )
                                self._models.unload_model(model_name, model_version)
                            except Exception:
                                logger.info(
                                    f"failed unloading model {model_name} of version {model_version} from TFS"
                                )
                                raise

                        # remove model from disk and references
                        if status in ["on-disk", "in-memory"]:
                            logger.info(
                                f"removing model references from memory and from disk for model {model_name} of version {model_version}"
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
                            kwargs={
                                "model_name": model_name,
                                "model_version": model_version,
                                "signature_key": self._determine_model_signature_key(model_name),
                            },
                        )
                    except Exception as e:
                        raise UserRuntimeException(
                            f"failed (re-)loading model {model_name} of version {model_version} (thread {td.get_ident()})",
                            str(e),
                        )

                    # run prediction
                    self._models.get_model(model_name, model_version, tag)
                    prediction = self._client.predict(model_input, model_name, model_version)

            return prediction

    def _load_model(
        self, model_path: str, model_name: str, model_version: str, signature_key: Optional[str]
    ) -> Any:
        """
        Loads model into TFS.
        Must only be used when caching enabled.
        """

        try:
            model_dir = os.path.split(model_path)[0]
            self._client.add_single_model(
                model_name, model_version, model_dir, signature_key, timeout=30.0, max_retries=3
            )
        except Exception as e:
            self._client.remove_single_model(model_name, model_version)
            raise

        return "loaded tensorflow model"

    def _remove_models(self, model_ids: List[str]) -> None:
        """
        Remove models from TFS.
        Must only be used when caching enabled.
        """
        logger.info(f"unloading models with model IDs {model_ids} from TFS")

        models = {}
        for model_id in model_ids:
            model_name, model_version = model_id.rsplit("-", maxsplit=1)
            if model_name not in models:
                models[model_name] = [model_version]
            else:
                models[model_name].append(model_version)

        model_names = []
        model_versions = []
        for model_name, versions in models.items():
            model_names.append(model_name)
            model_versions.append(versions)

        self._client.remove_models(model_names, model_versions)

    def _determine_model_signature_key(self, model_name: str) -> Optional[str]:
        """
        Determine what's the signature key for a given model from API spec.
        """
        if self._models_dir:
            return self._api_spec["predictor"]["models"]["signature_key"]
        return self._spec_models[model_name]["signature_key"]

    def _get_latest_model_version_from_tree(self, model_name: str, model_info: dict) -> str:
        """
        Get the highest version for a specific model name.
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
        When caching is disabled, the returned dictionary will be like in the following example:
        {
            ...
            "image-classifier-inception-1569014553": {
                "disk_path": "/mnt/model/image-classifier-inception/1569014553",
                "signature_def": {
                    "predict": {
                    "inputs": {
                        "images": {
                        "name": "images:0",
                        "dtype": "DT_FLOAT",
                        "tensorShape": {
                            "dim": [
                            {
                                "size": "-1"
                            },
                            {
                                "size": "-1"
                            },
                            {
                                "size": "-1"
                            },
                            {
                                "size": "3"
                            }
                            ]
                        }
                        }
                    },
                    "outputs": {
                        "classes": {
                        "name": "module_apply_default/InceptionV3/Logits/SpatialSqueeze:0",
                        "dtype": "DT_FLOAT",
                        "tensorShape": {
                            "dim": [
                            {
                                "size": "-1"
                            },
                            {
                                "size": "1001"
                            }
                            ]
                        }
                        }
                    },
                    "methodName": "tensorflow/serving/predict"
                    }
                },
                "signature_key": "predict",
                "input_signature": {
                    "images": {
                    "shape": [
                        -1,
                        -1,
                        -1,
                        3
                    ],
                    "type": "float32"
                    }
                },
                "timestamp": 1602025473
            }
            ...
        }

        Or when the caching is enabled, the following represents the kind of returned dictionary:
        {
            ...
            "image-classifier-inception": {
                "versions": [
                    "1569014553",
                    "1569014559"
                ],
                "timestamps": [
                    "1601668127",
                    "1601668120"
                ]
            }
            ...
        }
        """

        if not self._caching_enabled:
            # the models dictionary has another field for each key entry
            # called timestamp inserted by TFSAPIServingThreadUpdater thread
            return self._client.models
        else:
            models_info = self._models_tree.get_all_models_info()
            for model_name in models_info.keys():
                del models_info[model_name]["bucket"]
                del models_info[model_name]["model_paths"]
            return models_info

    @property
    def caching(self) -> bool:
        return self._caching_enabled
