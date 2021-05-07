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

from typing import List, Optional

import cortex_internal.consts
from cortex_internal.lib.model import find_all_s3_models
from cortex_internal.lib.type import handler_type_from_api_spec, PythonHandlerType


class CuratedModelResources:
    def __init__(self, curated_model_resources: List[dict]):
        """
        An example of curated_model_resources object:
        [
            {
                'path': 's3://cortex-examples/models/tensorflow/transformer/',
                'name': 'modelB',
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

    def get_field(self, field: str) -> List[str]:
        """
        Get a list of the values of each models' specified field.

        Args:
            field: name, path, signature_key or versions.

        Returns:
            A list with the specified value of each model.
        """
        return [model[field] for model in self._models]

    def get_versions_for(self, name: str) -> Optional[List[str]]:
        """
        Get versions for a given model name.

        Args:
            name: Name of the model (_cortex_default for handler:models:path) or handler:models:paths:name.

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

    def get_s3_model_names(self) -> List[str]:
        """
        Get S3 models as specified with handler:models:path or handler:models:paths.

        Returns:
            A list of names of all models available from the bucket(s).
        """
        return self.get_field("name")

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
    handler_type = handler_type_from_api_spec(api_spec)

    if handler_type == PythonHandlerType and api_spec["handler"]["multi_model_reloading"]:
        models_spec = api_spec["handler"]["multi_model_reloading"]
    elif handler_type != PythonHandlerType:
        models_spec = api_spec["handler"]["models"]
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

        model_resource["path"] = model["path"]
        _, versions, _, _, _, _ = find_all_s3_models(
            False, "", handler_type, [model_resource["path"]], [model_resource["name"]]
        )
        if model_resource["name"] not in versions:
            continue
        model_resource["versions"] = versions[model_resource["name"]]

        model_resources.append(model_resource)

    return CuratedModelResources(model_resources)
