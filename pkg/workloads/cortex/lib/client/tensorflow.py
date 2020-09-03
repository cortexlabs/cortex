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
from typing import Any, Optional, List

from cortex.lib.exceptions import UserRuntimeException, CortexException, UserException, WithBreak
from cortex.lib.model import (
    TensorFlowServingAPI,
    ModelsHolder,
    ModelsTree,
    LockedModelsTree,
    CuratedModelResources,
)
from cortex.lib.log import cx_logger
from cortex import consts

logger = cx_logger()


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
            tf_serving_url: Localhost URL to TF Serving container.
            api_spec: API configuration.

            models: Holding all models into memory. Only when processes_per_replica = 1.
            model_dir: Where the models are saved on disk. Only when processes_per_replica = 1.
            models_tree: A tree of the available models from upstream. Only when processes_per_replica = 1.
        """

        self._tf_serving_url = tf_serving_url
        self._api_spec = api_spec
        self._models = models
        self._models_tree = models_tree
        self._model_dir = model_dir

        self._spec_models = CuratedModelResources(api_spec["curated_model_resources"])

        if self._api_spec["predictor"]["models"]["dir"] is not None:
            self._models_dir = True
        else:
            self._models_dir = False
            self._spec_model_names = self._spec_models.get_field("name")

        if self._api_spec["predictor"]["processes_per_replica"] > 1:
            self._multiple_processes = True
        else:
            self._multiple_processes = False

        if self._models:
            self._models.set_callback("load", self._load_model)
            self._models.set_callback("remove", self._remove_models)

        self._client = TensorFlowServingAPI(tf_serving_url)

    def predict(
        self, model_input: Any, model_name: Optional[str] = None, model_version: str = "highest"
    ) -> dict:
        """
        Validate model_input, convert it to a Prediction Proto, and make a request to TensorFlow Serving.

        Args:
            model_input: Input to the model.
            model_name: Model to use when multiple models are deployed in a single API.

        Returns:
            dict: TensorFlow Serving response converted to a dictionary.
        """
        if model_version not in ["highest", "latest"] or not model_version.isnumeric():
            raise UserRuntimeException(
                "model_version must be either a parse-able numeric value or 'highest' or 'latest'"
            )

        if self._multiple_processes and model_version == "latest":
            raise UserRuntimeException(
                "model_version must be either a parse-able numberic value or 'highest'",
                "cannot be 'latest' when processes_per_replica > 1 and TensorFlowPredictor is used",
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
        if self._models_dir and model_name is None:
            raise UserRuntimeException("model_name was not specified")

        return self._run_inference(model_input, model_name, model_version)

    def _run_inference(self, model_input: Any, model_name: str, model_version: str) -> dict:
        """
        When processes_per_replica = 1, check/load model and make prediction.
        When processes_per_replica > 1, attempt to make prediction regardless.

        Args:
            model_input: Input to the model.
            model_name: Name of the model, as it's specified in predictor:models:paths or in the other case as they are named on disk.
            model_version: Version of the model, as it's found on disk. Can also infer the version number from "latest" and "highest" tags.

        Exceptions:
            RuntimeError: if another thread tried to load the model at the very same time.

        Returns:
            The prediction.
        """

        model = None
        tag = ""
        tags = []
        if model_version in ["latest", "highest"]:
            tag = model_version
            tags = ["latest", "highest"]

        if self._multiple_processes:
            # determine model version
            if tag == "highest":
                versions = self._client.poll_available_models(model_name)
                if len(versions) == 0:
                    raise UserException(
                        "model '{}' accessed with tag '{}' couldn't be found".format(
                            model_name, tag
                        )
                    )
                model_version = max(versions)
            model_id = model_name + "-" + model_version

            return self._client.predict(model_input, model_name, model_version)

        if not self._multiple_processes:
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
                    raise WithBreak

                # retrieve model tree's metadata
                upstream_model = self._models_tree[model_id]
                current_upstream_ts = upstream_model["timestamp"]

            if not available_model:
                if tag == "":
                    raise UserException(
                        "model '{}' of version '{}' couldn't be found".format(
                            model_name, model_version
                        )
                    )
                raise UserException(
                    "model '{}' accessed with tag '{}' couldn't be found".format(model_name, tag)
                )

            # grab shared access to models holder and retrieve model
            update_model = False
            prediction = None
            with LockedModel(self._models, "r", model_name, model_version):
                status, upstream_ts = self._models.has_model(model_name, model_version)
                if status in ["not-available", "on-disk"] or (
                    status != "not-available" and upstream_ts < current_upstream_ts
                ):
                    update_model = True
                    raise WithBreak

                # run prediction
                self._models.get_model(model_name, model_version, tag)
                prediction = self._client.predict(model_input, model_name, model_version)

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
                            raise WithBreak
                        current_upstream_ts = date.timestamp()

                    # load model
                    try:
                        self._models.load_model(
                            model_name,
                            model_version,
                            current_upstream_ts,
                            tags,
                            kwargs={
                                "model_name": model_name,
                                "model_version": model_version,
                                "signature_key": self._determine_model_signature_key(model_name),
                            },
                        )
                    except Exception:
                        raise WithBreak

                    # run prediction
                    self._models.get_model(model_name, model_version, tag)
                    prediction = self._client.predict(model_input, model_name, model_version)

            if prediction:
                return prediction
            if tag == "":
                raise UserException(
                    "could not run prediction on model '{}' of version '{}' because the model couldn't be loaded or isn't available".format(
                        model_name, model_version
                    )
                )
            raise UserException(
                "could not run prediction on model '{}' accessed with '{}' tag because the model couldn't be loaded or isn't available".format(
                    model_name, tag
                )
            )

    def _load_model(
        self, model_path: str, model_name: str, model_version: str, signature_key: Optional[str]
    ) -> Any:
        """
        Loads model into TFS.
        Must only be used when processes_per_replica = 1.
        """

        try:
            self._client.add_single_model(
                model_name, model_version, model_path, signature_key, timeout=30.0
            )
        except Exception as e:
            self._client.remove_single_model(model_name, model_version)
            raise

        return "loaded tensorflow model"

    def _remove_models(self, model_ids: List[str]) -> None:
        """
        Remove models from TFS.
        Must only be used when processes_per_replica = 1.
        """

        models = {}
        for model_id in model_ids:
            model_name, model_version = model_id.rsplit("-")
            if model_name not in models:
                models[model_name] = [model_version]
            else:
                models[model_name].append(model_version)

        model_names = []
        model_versions = []
        for model_name, versions in models:
            model_names.append(model_name)
            model_versions.append(versions)

        self._client.remove_models(model_names, model_versions)

    def _determine_model_signature_key(self, model_name: str) -> Optional[str]:
        """
        Determine what's the signature key for a given model from API spec.
        """

        signature_key = None

        if self._models_dir:
            signature_key = self._api_spec["predictor"]["models"]["signature_key"]
        else:
            signature_key = self._spec_model_names[model_name]["signature_key"]

        return signature_key

    def _get_model_version_from_tree(self, model_name: str, tag: str, model_info: dict) -> str:
        """
        Get the version for a specific model name based on the version tag - either "latest" or "highest".
        Must only be used when processes_per_replica = 1.
        """

        if tag not in ["latest", "highest"]:
            raise ValueError("invalid tag; must be either 'latest' or 'highest'")

        versions, timestamps = model_info["versions"], model_info["timestamps"]
        if tag == "latest":
            index = timestamps.index(max(timestamps))
            return versions[index]
        else:
            return max(versions)

    @property
    def stub(self):
        return self._stub

    @property
    def input_signatures(self):
        return self._input_signatures
