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
from typing import List, Optional

import cortex_internal.consts
from cortex_internal.lib.model import find_all_cloud_models
from cortex_internal.lib.type import predictor_type_from_api_spec, PythonPredictorType


class CuratedModelResources:
    def __init__(self, curated_model_resources: List[dict]):
        """
        An example of curated_model_resources object:
        [
            {
                'path': 's3://cortex-examples/models/tensorflow/transformer/',
                'name': 'modelB',
                's3_path': True,
                'gs_path': False,
                'local_path': False,
                'signature_key': None,
                'versions': [1554540232]
            },
            ...
        ]
        """
        self._models = curated_model_resources

        for res in self._models:
            if not res["versions"]:
                res["versions"] = []
            else:
                res["versions"] = [str(version) for version in res["versions"]]

    def is_local(self, name: str) -> Optional[bool]:
        """
        Checks if the model has been made available from the local disk.

        Note: Only required for ONNX file paths.

        Args:
            name: Name of the model as specified in predictor:models:paths:name or if a single model is specified, _cortex_default.

        Returns:
            If the model is local. None if the model wasn't found.
        """
        for model in self._models:
            if model["name"] == name:
                return model["local_path"]
        return None

    def get_field(self, field: str) -> List[str]:
        """
        Get a list of the values of each models' specified field.

        Args:
            field: name, s3_path, gs_path, local_path, signature_key or versions.

        Returns:
            A list with the specified value of each model.
        """
        return [model[field] for model in self._models]

    def get_versions_for(self, name: str) -> Optional[List[str]]:
        """
        Get versions for a given model name.

        Args:
            name: Name of the model (_cortex_default for predictor:models:path) or predictor:models:paths:name.

        Returns:
            Versions for a given model. None if the model wasn't found.
        """
        versions = []
        model_not_found = True
        for i, _ in enumerate(self._models):
            if self._models[i]["name"] == name:
                versions = self._models[i]["versions"]
                model_not_found = False
                break

        if model_not_found:
            return None
        return [str(version) for version in versions]

    def get_local_model_names(self) -> List[str]:
        """
        Get locally-provided models as specified with predictor:models:path, predictor:models:paths or predictor:models:dir.

        Note: Only required for ONNX file paths.

        Returns:
            A list of names of all local models.
        """
        local_model_names = []
        for model_name in self.get_field("name"):
            if self.is_local(model_name):
                local_model_names.append(model_name)

        return local_model_names

    def get_cloud_model_names(self) -> List[str]:
        """
        Get cloud-provided models as specified with predictor:models:path or predictor:models:paths.

        Returns:
            A list of names of all models available from the cloud bucket(s).
        """
        cloud_model_names = []
        for model_name in self.get_field("name"):
            if not self.is_local(model_name):
                cloud_model_names.append(model_name)

        return cloud_model_names

    def __getitem__(self, name: str) -> dict:
        """
        Gets the model resource for a given model name.
        """
        for model in self._models:
            if model["name"] == name:
                return model

        raise KeyError(f"model resource {name} does not exit")

    def __contains__(self, name: str) -> bool:
        """
        Checks if there's a model resource whose name is the provided one.
        """
        try:
            self[name]
            return True
        except KeyError:
            return False


def get_models_from_api_spec(
    api_spec: dict, model_dir: str = "/mnt/model"
) -> CuratedModelResources:
    """
    Only effective for models:path, models:paths or for models:dir fields when the dir is a local path.
    It does not apply for when models:dir field is set to an S3 model path.
    """
    predictor_type = predictor_type_from_api_spec(api_spec)

    if predictor_type == PythonPredictorType and api_spec["predictor"]["multi_model_reloading"]:
        models_spec = api_spec["predictor"]["multi_model_reloading"]
    elif predictor_type != PythonPredictorType:
        models_spec = api_spec["predictor"]["models"]
    else:
        return CuratedModelResources([])

    if not models_spec["path"] and not models_spec["paths"]:
        return CuratedModelResources([])

    # for models.path
    models = []
    if models_spec["path"]:
        model = {
            "name": cortex_internal.consts.SINGLE_MODEL_NAME,
            "path": models_spec["path"],
            "signature_key": models_spec["signature_key"],
        }
        models.append(model)

    # for models.paths
    if models_spec["paths"]:
        for model in models_spec["paths"]:
            models.append(
                {
                    "name": model["name"],
                    "path": model["path"],
                    "signature_key": model["signature_key"],
                }
            )

    # building model resources for models.path or models.paths
    model_resources = []
    for model in models:
        model_resource = {}
        model_resource["name"] = model["name"]

        if not model["signature_key"]:
            model_resource["signature_key"] = models_spec["signature_key"]
        else:
            model_resource["signature_key"] = model["signature_key"]

        ends_as_file_path = model["path"].endswith(".onnx")
        if ends_as_file_path and os.path.exists(
            os.path.join(model_dir, model_resource["name"], "1", os.path.basename(model["path"]))
        ):
            model_resource["is_file_path"] = True
            model_resource["s3_path"] = False
            model_resource["gcs_path"] = False
            model_resource["local_path"] = True
            model_resource["versions"] = []
            model_resource["path"] = os.path.join(
                model_dir, model_resource["name"], "1", os.path.basename(model["path"])
            )
            model_resources.append(model_resource)
            continue
        model_resource["is_file_path"] = False

        model_resource["s3_path"] = model["path"].startswith("s3://")
        model_resource["gcs_path"] = model["path"].startswith("gs://")
        model_resource["local_path"] = (
            not model_resource["s3_path"] and not model_resource["gcs_path"]
        )

        if model_resource["s3_path"] or model_resource["gcs_path"]:
            model_resource["path"] = model["path"]
            _, versions, _, _, _, _, _ = find_all_cloud_models(
                False, "", predictor_type, [model_resource["path"]], [model_resource["name"]]
            )
            if model_resource["name"] not in versions:
                continue
            model_resource["versions"] = versions[model_resource["name"]]
        else:
            model_resource["path"] = os.path.join(model_dir, model_resource["name"])
            model_resource["versions"] = os.listdir(model_resource["path"])

        model_resources.append(model_resource)

    # building model resources for models.dir
    if (
        models_spec
        and models_spec["dir"]
        and not models_spec["dir"].startswith("s3://")
        and not models_spec["dir"].startswith("gs://")
    ):
        for model_name in os.listdir(model_dir):
            model_resource = {}
            model_resource["name"] = model_name
            model_resource["s3_path"] = False
            model_resource["gcs_path"] = False
            model_resource["local_path"] = True
            model_resource["signature_key"] = models_spec["signature_key"]
            model_resource["path"] = os.path.join(model_dir, model_name)
            model_resource["versions"] = os.listdir(model_resource["path"])
            model_resources.append(model_resource)

    return CuratedModelResources(model_resources)
