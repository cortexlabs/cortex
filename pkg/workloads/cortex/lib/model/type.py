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

from typing import List, Optional


class CuratedModelResources:
    def __init__(self, curated_model_resources: List[dict]):
        """
        curated_model_resources must have the format enforced by the CLI's validation process of cortex.yaml.
        curated_model_resources is an identical copy of pkg.type.spec.api.API.CuratedModelResources.

        An example of curated_model_resources object:
        [
            {
                'model_path': 's3://cortex-0/models/tensorflow/transformer/',
                'name': 'modelB',
                's3_path': True,
                'signature_key': None,
                'versions': [1554540232]
            },
            ...
        ]
        """
        self._models = curated_model_resources

        for res in self._models:
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
                return not model["s3_path"]
        return None

    def get_field(self, field: str) -> List[str]:
        """
        Get a list of the values of each models' specified field.

        Args:
            field: name, s3_path, signature_key or versions.

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

    def get_s3_model_names(self) -> List[str]:
        """
        Get S3-provided models as specified with predictor:model_path, predictor:models:paths or predictor:models:dir.

        Returns:
            A list of names of all models available from S3.
        """
        s3_model_names = []
        for model_name in self.get_field("name"):
            if not self.is_local(model_name):
                s3_model_names.append(model_name)

        return s3_model_names

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
