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
import datetime
import shutil
import threading as td
from typing import List, Dict, Any, Tuple, Callable, AbstractSet

from cortex.lib.log import cx_logger as logger
from cortex.lib.concurrency import ReadWriteLock
from cortex.lib.exceptions import WithBreak


class ModelsTree:
    """
    Model tree for S3-provided models.
    """

    def __init__(self):
        self.models = {}
        self._locks = {}
        self._create_lock = td.RLock()
        self._removable = set()

    def acquire(self, mode: str, model_name: str, model_version: str) -> None:
        """
        Acquire shared/exclusive (R/W) access for a specific model. Use this when multiple threads are used.

        Args:
            mode: "r" for read lock, "w" for write lock.
            model_name: The name of the model.
            model_version: The version of the model.
        """
        model_id = f"{model_name}-{model_version}"

        if not model_id in self._locks:
            lock = ReadWriteLock()
            lock_id = id(lock)
            self._create_lock.acquire()
            if model_id not in self._locks:
                self._locks[model_id] = lock
            self._create_lock.release()

        self._locks[model_id].acquire(mode)

    def release(self, mode: str, model_name: str, model_version: str) -> None:
        """
        Release shared/exclusive (R/W) access for a specific model. Use this when multiple threads are used.

        Args:
            mode: "r" for read lock, "w" for write lock.
            model_name: The name of the model.
            model_version: The version of the model.
        """
        model_id = f"{model_name}-{model_version}"
        self._locks[model_id].release(mode)

    def update_models(
        self,
        model_names: List[str],
        model_versions: Dict[str, List[str]],
        model_paths: List[str],
        sub_paths: List[List[str]],
        timestamps: List[List[datetime.datetime]],
        bucket_names: List[str],
    ) -> Tuple[AbstractSet[str], AbstractSet[str]]:
        """
        Updates the model tree with the latest from the upstream and removes stale models.

        Locking is not required. Locking is already done within the method.

        Args:
            model_names: The unique names of the models as discovered in models:dir or specified in models:paths.
            model_versions: The detected versions of each model. If the list is empty, then version "1" should be assumed. The dictionary keys represent the models' names.
            model_paths: S3 model paths to each model.
            sub_paths: A list of filepaths lists for each file of each model.
            timestamps: When was each versioned model updated the last time on the upstream. When no versions are passed, a timestamp is still expected.
            bucket_names: A list with the bucket_names required for each model.

        Returns:
            The loaded model IDs ("<model-name>-<model-version") that haven't been found in the passed parameters.
            Which model IDs have been updated. If these model IDs are in memory or on disk already, then they should get updated as well.

        Also sets an info attribute which might look like this:
        {
            "<model-name>": ,
        }
        And where "versions" represents the available versions of a model <model-name> and each "timestamps" element is the corresponding
        last-edit time of each versioned model.
        """

        current_model_ids = set()
        updated_model_ids = set()
        for idx in range(len(model_names)):
            model_name = model_names[idx]

            if len(model_versions[model_name]) == 0:
                model_id = f"{model_name}-1"
                with LockedModelsTree(self, "w", model_name, "1"):
                    updated = self.update_model(
                        bucket_names[idx],
                        model_name,
                        "1",
                        model_paths[idx],
                        sub_paths[idx],
                        timestamps[idx][0],
                        True,
                    )
                current_model_ids.add(model_id)
                if updated:
                    updated_model_ids.add(model_id)

            for v_idx, model_version in enumerate(model_versions[model_name]):
                model_id = f"{model_name}-{model_version}"
                with LockedModelsTree(self, "w", model_name, model_version):
                    updated = self.update_model(
                        bucket_names[idx],
                        model_name,
                        model_version,
                        os.path.join(model_paths[idx], model_version) + "/",
                        sub_paths[idx],
                        timestamps[idx][v_idx],
                        True,
                    )
                current_model_ids.add(model_id)
                if updated:
                    updated_model_ids.add(model_id)

        old_model_ids = set(self.models.keys()) - current_model_ids
        aux_old_model_ids = old_model_ids

        for old_model_id in old_model_ids:
            model_name, model_version = old_model_id.rsplit("-", maxsplit=1)
            with LockedModelsTree(self, "w", model_name, model_version):
                if old_model_id in self._removable:
                    del self.models[old_model_id]
                    self._removable = self._removable - set([old_model_id])
                    old_model_ids = old_model_ids - set([old_model_id])

        return aux_old_model_ids, updated_model_ids

    def update_model(
        self,
        bucket: str,
        model_name: str,
        model_version: str,
        model_path: str,
        sub_paths: List[str],
        timestamp: datetime.datetime,
        tree_removable: bool,
    ) -> None:
        """
        Updates the model tree with the given model.

        Locking is required.

        Args:
            bucket: The S3 bucket on which the model is stored.
            model_name: The unique name of the model as discovered in models:dir or specified in models:paths.
            model_version: A detected version of the model.
            model_path: The model path to the versioned model.
            sub_paths: A list of filepaths for each file of the model.
            timestamp: When was the model path updated the last time.
            tree_removable: If update_models method is allowed to remove the model.

        Returns:
            True if the model wasn't in the tree or if the timestamp is newer. False otherwise.
        """

        model_id = f"{model_name}-{model_version}"
        changed = False
        if model_id not in self.models:
            changed = True
        elif self.models[model_id]["timestamp"] < timestamp:
            changed = True

        if changed or model_id in self.models:
            self.models[model_id] = {
                "bucket": bucket,
                "path": model_path,
                "sub_paths": sub_paths,
                "timestamp": timestamp,
            }
            if tree_removable:
                self._removable.add(model_id)
            else:
                self._removable = self._removable - set([model_id])

        return changed

    def model_info(self, model_name: str) -> dict:
        """
        Gets model info about the available versions and model timestamps.

        Locking is not required.

        Returns:
            A dict with keys "bucket", "model_paths, "versions" and "timestamps".
            "model_paths" contains the S3 prefixes of each versioned model, "versions" represents the available versions of the model,
            and each "timestamps" element is the corresponding last-edit time of each versioned model.

            Empty lists are returned if the model is not found.

        Example of returned dictionary for model_name.
        ```json
        {
            "bucket": "bucket-0",
            "model_paths": ["modelA/1", "modelA/4", "modelA/7", ...],
            "versions": [1,4,7, ...],
            "timestamps": [12884999, 12874449, 12344931, ...]
        }
        ```
        """

        info = {
            "model_paths": [],
            "versions": [],
            "timestamps": [],
        }

        models = self.models.copy()
        for model_id in models:
            _model_name, model_version = model_id.rsplit("-", maxsplit=1)
            if _model_name == model_name:
                if "bucket" not in info:
                    info["bucket"] = models[model_id]["bucket"]
                info["model_paths"] += [os.path.join(models[model_id]["path"], model_version)]
                info["versions"] += [model_version]
                info["timestamps"] += [models[model_id]["timestamp"]]

        return info

    def get_model_names(self) -> List[str]:
        """
        Gets the available model names.

        Locking is not required.

        Returns:
            List of all model names.
        """
        model_names = set()
        models = self.models.copy()
        for model_id in models:
            model_name = model_id.rsplit("-", maxsplit=1)[0]
            model_names.add(model_name)

        return list(model_names)

    def __getitem__(self, model_id: str) -> dict:
        """
        Each value of a key (model ID) is a dictionary with the following format:
        {
            "bucket": <bucket-of-the-model>,
            "path": <path-of-the-model>,
            "sub_paths": <sub-path-of-each-file-of-the-model>,
            "timestamp": <when-was-the-model-last-modified>
        }

        Locking is required.
        """
        return self.models[model_id].copy()

    def __contains__(self, model_id: str) -> bool:
        """
        Each value of a key (model ID) is a dictionary with the following format:
        {
            "bucket": <bucket-of-the-model>,
            "path": <path-of-the-model>,
            "sub_paths": <sub-path-of-each-file-of-the-model>,
            "timestamp": <when-was-the-model-last-modified>
        }

        Locking is required.
        """
        return model_id in self.models


class LockedModelsTree:
    """
    When acquiring shared/exclusive (R/W) access to a model resource (model name + version).

    Locks just for a specific model. Apply read lock when granting shared access or write lock when it's exclusive access (for adding/removing operations).

    The context manager can be exited by raising cortex.lib.exceptions.WithBreak.
    """

    def __init__(self, tree: ModelsTree, mode: str, model_name: str, model_version: str):
        """
        mode can be "r" for read or "w" for write.
        """
        self._tree = tree
        self._mode = mode
        self._model_name = model_name
        self._model_version = model_version

    def __enter__(self):
        self._tree.acquire(self._mode, self._model_name, self._model_version)
        return self

    def __exit__(self, exc_type, exc_value, traceback) -> bool:
        self._tree.release(self._mode, self._model_name, self._model_version)

        if exc_value is not None and exc_type is not WithBreak:
            return False
        return True
