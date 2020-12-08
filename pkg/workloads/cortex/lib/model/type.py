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
from typing import List, Optional

import cortex.consts
from cortex.lib.model import find_all_cloud_models
from cortex.lib.type import predictor_type_from_api_spec


class CuratedModelResources:
    def __init__(self, curated_model_resources: List[dict]):
        """
        An example of curated_model_resources object:
        [
            {
                'model_path': 's3://cortex-examples/models/tensorflow/transformer/',
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
            name: Name of the model (_cortex_default for predictor:model_path) or predictor:models:paths:name.

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
        Get locally-provided models as specified with predictor:model_path, predictor:models:paths or predictor:models:dir.

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
        Get cloud-provided models as specified with predictor:model_path or predictor:models:paths.

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
    Only effective for predictor:model_path, predictor:models:paths or for predictor:models:dir when the dir is a local path.
    It does not apply for when predictor:models:dir is set to an S3 model path.
    """

    predictor = api_spec["predictor"]

    if not predictor["model_path"] and not predictor["models"]:
        return CuratedModelResources([])

    predictor_type = predictor_type_from_api_spec(api_spec)

    # for predictor.model_path
    models = []
    if predictor["model_path"]:
        model = {
            "name": cortex.consts.SINGLE_MODEL_NAME,
            "model_path": predictor["model_path"],
            "signature_key": predictor["signature_key"],
        }
        models.append(model)

    # for predictor.models.paths
    if predictor["models"] and predictor["models"]["paths"]:
        for model in predictor["models"]["paths"]:
            models.append(
                {
                    "name": model["name"],
                    "model_path": model["model_path"],
                    "signature_key": model["signature_key"],
                }
            )

    # building model resources for predictor.model_path or predictor.models.paths
    model_resources = []
    for model in models:
        model_resource = {}
        model_resource["name"] = model["name"]
        model_resource["s3_path"] = model["model_path"].startswith("s3://")
        model_resource["gcs_path"] = model["model_path"].startswith("gs://")
        model_resource["local_path"] = (
            not model_resource["s3_path"] and not model_resource["gcs_path"]
        )

        if not model["signature_key"] and predictor["models"]:
            model_resource["signature_key"] = predictor["models"]["signature_key"]
        else:
            model_resource["signature_key"] = model["signature_key"]

        if model_resource["s3_path"] or model_resource["gcs_path"]:
            model_resource["model_path"] = model["model_path"]
            _, versions, _, _, _, _, _ = find_all_cloud_models(
                False, "", predictor_type, [model_resource["model_path"]], [model_resource["name"]]
            )
            if model_resource["name"] not in versions:
                continue
            model_resource["versions"] = versions[model_resource["name"]]
        else:
            model_resource["model_path"] = os.path.join(model_dir, model_resource["name"])
            model_resource["versions"] = os.listdir(model_resource["model_path"])

        model_resources.append(model_resource)

    # building model resources for predictor.models.dir
    if (
        predictor["models"]
        and predictor["models"]["dir"]
        and not predictor["models"]["dir"].startswith("s3://")
        and not predictor["models"]["dir"].startswith("gs://")
    ):
        for model_name in os.listdir(model_dir):
            model_resource = {}
            model_resource["name"] = model_name
            model_resource["s3_path"] = False
            model_resource["gcs_path"] = False
            model_resource["local_path"] = True
            model_resource["signature_key"] = predictor["models"]["signature_key"]
            model_resource["model_path"] = os.path.join(model_dir, model_name)
            model_resource["versions"] = os.listdir(model_resource["model_path"])
            model_resources.append(model_resource)

    return CuratedModelResources(model_resources)
