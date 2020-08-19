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

from typing import List, Any, Tuple, Callable, AbstractSet
from cortex.lib.concurrency import ReadWriteLock
from cortex.lib.exceptions import WithBreak

import os
import time
import shutil
import threading as td


class ModelsTree:
    def __init__(self):
        self._models = {}
        self._lock = ReadWriteLock()

    def acquire(self, mode: str) -> None:
        """
        Acquire lock on the model tree.

        Args:
            mode: "r" for read lock, "w" for write lock.
        """
        self._lock.acquire(mode)

    def release(self, mode: str) -> None:
        """
        Release lock on the model tree.

        Args:
            mode: "r" for read lock, "w" for write lock.
        """
        self._lock.release(mode)

    def update_models(
        self,
        model_names: List[str],
        model_versions: List[List[str]],
        model_paths: List[str],
        timestamps: List[List[int]],
    ) -> Tuple[AbstractSet[str], AbstractSet[str]]:
        """
        Updates the model tree with the latest from the upstream.

        Args:
            model_names: The unique names of the models as discovered in models:dir or specified in models:paths.
            model_versions: The detected versions of each model. "none" if no version is found.
            model_paths: S3 model paths to each model.
            timestamps: When was each versioned model updated the last time on the upstream.
        
        Returns:
            The model IDs ("<model-name>-<model-version") that haven't been found in the passed parameters.
            Which model IDs have been updated. If these model IDs are in memory or on disk already, then they should get updated as well.
        """

        current_model_ids = set()
        updated_model_ids = set()
        for idx in range(len(model_names)):
            model_name = model_names[idx]
            if len(model_versions[idx]) == 0:
                model_id = f"{model_name}-none"
                updated = self.update_model(
                    model_name, "none", model_paths[idx], timestamps[idx][0]
                )
                current_model_ids.add(model_id)
                if updated:
                    updated_model_ids.add(model_id)
            for model_version in model_versions[idx]:
                model_id = f"{model_name}-{model_version}"
                updated = self.update_model(
                    model_name,
                    model_version,
                    os.path.join(model_paths[idx], model_version),
                    timestamps[idx][model_version],
                )
                current_model_ids.add(model_id)
                if updated:
                    updated_model_ids.add(model_id)

        old_model_ids = set(self._models.keys()) - current_model_ids
        for old_model_id in old_model_ids:
            del self._models[old_model_id]

        return old_model_ids, updated_model_ids

    def update_model(
        self, model_name: str, model_version: str, model_path: str, timestamp: int,
    ) -> None:
        """
        Updates the model tree with the given model.

        Args:
            model_name: The unique name of the model as discovered in models:dir or specified in models:paths.
            model_version: A detected version of the model. "none" if no version is found.
            model_paths: S3 model path to the versioned model.
            timestamp: When was the model path updated the last time.

        Returns:
            True if the model wasn't in the tree or if the timestamp is newer. False otherwise.
        """

        model_id = f"{model_name}-{model_version}"
        changed = False
        if model_id not in self._models:
            changed = True
        elif self._models[model_id]["timestamp"] < timestamp:
            changed = True

        if changed or model_id in self._models:
            self._models[model_id] = {
                "path": model_path,
                "timestamp": timestamp,
            }

        return changed


class LockedModelsTree:
    """
    When acquiring R/W access to a model resource (model name + version).

    Locks the entire tree. When receiving requests, the read lock shall be applied.
    When updating the tree in the cron, the write lock shall be applied.

    The context manager can be exited by raising cortex.lib.exceptions.WithBreak.
    """

    def __init__(self, tree: ModelsTree, mode: str):
        self._tree = tree
        self._mode = mode
        self._lock = ReadWriteLock()

    def __enter__(self):
        self._lock.acquire(self._mode)
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self._lock.release(self._mode)

        if exc_value is not None and exc_type is not WithBreak:
            raise exc_type(exc_value).with_traceback(traceback)
