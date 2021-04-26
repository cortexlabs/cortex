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

import datetime
import itertools
import os
import threading as td
from typing import List, Dict, Tuple, AbstractSet

from cortex_internal.lib import util
from cortex_internal.lib.concurrency import ReadWriteLock
from cortex_internal.lib.exceptions import CortexException, WithBreak
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.model.validation import (
    validate_models_dir_paths,
    validate_model_paths,
    ModelVersion,
)
from cortex_internal.lib.storage import S3
from cortex_internal.lib.type import HandlerType

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


class ModelsTree:
    """
    Model tree for S3 models.
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
            bucket_names: A list with the bucket_names required for each model. Empty elements if no bucket is used.

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

        for old_model_id in old_model_ids:
            model_name, model_version = old_model_id.rsplit("-", maxsplit=1)
            if old_model_id not in self._removable:
                continue
            with LockedModelsTree(self, "w", model_name, model_version):
                del self.models[old_model_id]
                self._removable = self._removable - set([old_model_id])

        return old_model_ids, updated_model_ids

    def update_model(
        self,
        bucket: str,
        model_name: str,
        model_version: str,
        model_path: str,
        sub_paths: List[str],
        timestamp: datetime.datetime,
        removable: bool,
    ) -> bool:
        """
        Updates the model tree with the given model.

        Locking is required.

        Args:
            bucket: The bucket on which the model is stored. Empty if there's no bucket.
            model_name: The unique name of the model as discovered in models:dir or specified in models:paths.
            model_version: A detected version of the model.
            model_path: The model path to the versioned model.
            sub_paths: A list of filepaths for each file of the model.
            timestamp: When was the model path updated the last time.
            removable: If update_models method is allowed to remove the model.

        Returns:
            True if the model wasn't in the tree or if the timestamp is newer. False otherwise.
        """

        model_id = f"{model_name}-{model_version}"
        has_changed = False
        if model_id not in self.models:
            has_changed = True
        elif self.models[model_id]["timestamp"] < timestamp:
            has_changed = True

        if has_changed or model_id in self.models:
            self.models[model_id] = {
                "bucket": bucket,
                "path": model_path,
                "sub_paths": sub_paths,
                "timestamp": timestamp,
            }
            if removable:
                self._removable.add(model_id)
            else:
                self._removable = self._removable - set([model_id])

        return has_changed

    def model_info(self, model_name: str) -> dict:
        """
        Gets model info about the available versions and model timestamps.

        Locking is not required.

        Returns:
            A dict with keys "bucket", "model_paths, "versions" and "timestamps".
            "model_paths" contains the s3 prefixes of each versioned model, "versions" represents the available versions of the model,
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

        # to ensure atomicity
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

        # to ensure atomicity
        models = self.models.copy()
        for model_id in models:
            model_name = model_id.rsplit("-", maxsplit=1)[0]
            model_names.add(model_name)

        return list(model_names)

    def get_all_models_info(self) -> dict:
        """
        Gets model info about the available versions and model timestamps.

        Locking is not required.

        It's like model_info method, but for all model names.

        Example of returned dictionary.
        ```json
        {
            ...
            "modelA": {
                "bucket": "bucket-0",
                "model_paths": ["modelA/1", "modelA/4", "modelA/7", ...],
                "versions": ["1","4","7", ...],
                "timestamps": [12884999, 12874449, 12344931, ...]
            }
            ...
        }
        ```
        """

        models_info = {}
        # to ensure atomicity
        models = self.models.copy()

        # extract model names
        model_names = set()
        for model_id in models:
            model_name = model_id.rsplit("-", maxsplit=1)[0]
            model_names.add(model_name)
        model_names = list(model_names)

        # build models info dictionary
        for model_name in model_names:
            model_info = {
                "model_paths": [],
                "versions": [],
                "timestamps": [],
            }
            for model_id in models:
                _model_name, model_version = model_id.rsplit("-", maxsplit=1)
                if _model_name == model_name:
                    if "bucket" not in model_info:
                        model_info["bucket"] = models[model_id]["bucket"]
                    model_info["model_paths"] += [
                        os.path.join(models[model_id]["path"], model_version)
                    ]
                    model_info["versions"] += [model_version]
                    model_info["timestamps"] += [int(models[model_id]["timestamp"].timestamp())]

            models_info[model_name] = model_info

        return models_info

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

    The context manager can be exited by raising cortex_internal.lib.exceptions.WithBreak.
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


def find_all_s3_models(
    is_dir_used: bool,
    models_dir: str,
    handler_type: HandlerType,
    s3_paths: List[str],
    s3_model_names: List[str],
) -> Tuple[
    List[str],
    Dict[str, List[str]],
    List[str],
    List[List[str]],
    List[List[datetime.datetime]],
    List[str],
    List[str],
]:
    """
    Get updated information on all models that are currently present on the S3 upstreams.
    Information on the available models, versions, last edit times, the subpaths of each model, and so on.

    Args:
        is_dir_used: Whether handler:models:dir is used or not.
        models_dir: The value of handler:models:dir in case it's present. Ignored when not required.
        handler_type: The handler type.
        s3_paths: The S3 model paths as they are specified in handler:models:path/handler:models:paths/handler:models:dir is used. Ignored when not required.
        s3_model_names: The S3 model names as they are specified in handler:models:paths:name when handler:models:paths is used or the default name of the model when handler:models:path is used. Ignored when not required.

    Returns: The tuple with the following elements:
        model_names - a list with the names of the models (i.e. bert, gpt-2, etc) and they are unique
        versions - a dictionary with the keys representing the model names and the values being lists of versions that each model has.
          For non-versioned model paths ModelVersion.NOT_PROVIDED, the list will be empty.
        model_paths - a list with the prefix of each model.
        sub_paths - a list of filepaths lists for each file of each model.
        timestamps - a list of timestamps lists representing the last edit time of each versioned model.
        bucket_names - a list of the bucket names of each model.
    """

    # validate models stored in S3 that were specified with handler:models:dir field
    if is_dir_used:
        bucket_name, models_path = S3.deconstruct_s3_path(models_dir)
        client = S3(bucket_name)

        sub_paths, timestamps = client.search(models_path)

        model_paths, ooa_ids = validate_models_dir_paths(sub_paths, handler_type, models_path)
        model_names = [os.path.basename(model_path) for model_path in model_paths]

        model_paths = [
            model_path for model_path in model_paths if os.path.basename(model_path) in model_names
        ]
        model_paths = [
            model_path + "/" * (not model_path.endswith("/")) for model_path in model_paths
        ]

        bucket_names = len(model_paths) * [bucket_name]
        sub_paths = len(model_paths) * [sub_paths]
        timestamps = len(model_paths) * [timestamps]

    # validate models stored in S3 that were specified with handler:models:paths field
    if not is_dir_used:
        sub_paths = []
        ooa_ids = []
        model_paths = []
        model_names = []
        timestamps = []
        bucket_names = []
        for idx, path in enumerate(s3_paths):
            if S3.is_valid_s3_path(path):
                bucket_name, model_path = S3.deconstruct_s3_path(path)
                client = S3(bucket_name)
            else:
                continue

            sb, model_path_ts = client.search(model_path)
            try:
                ooa_ids.append(validate_model_paths(sb, handler_type, model_path))
            except CortexException:
                continue
            model_paths.append(model_path)
            model_names.append(s3_model_names[idx])
            bucket_names.append(bucket_name)
            sub_paths += [sb]
            timestamps += [model_path_ts]

    # determine the detected versions for each s3 model
    # if the model was not versioned, then leave the version list empty
    versions = {}
    for model_path, model_name, model_ooa_ids, bucket_sub_paths in zip(
        model_paths, model_names, ooa_ids, sub_paths
    ):
        if ModelVersion.PROVIDED not in model_ooa_ids:
            versions[model_name] = []
            continue

        model_sub_paths = [os.path.relpath(sub_path, model_path) for sub_path in bucket_sub_paths]
        model_versions_paths = [path for path in model_sub_paths if not path.startswith("../")]
        model_versions = [
            util.get_leftmost_part_of_path(model_version_path)
            for model_version_path in model_versions_paths
        ]
        model_versions = list(set(model_versions))
        versions[model_name] = model_versions

    # pick up the max timestamp for each versioned model
    aux_timestamps = []
    for model_path, model_name, bucket_sub_paths, sub_path_timestamps in zip(
        model_paths, model_names, sub_paths, timestamps
    ):
        model_ts = []
        if len(versions[model_name]) == 0:
            masks = list(
                map(
                    lambda x: x.startswith(model_path + "/" * (model_path[-1] != "/")),
                    bucket_sub_paths,
                )
            )
            model_ts = [max(itertools.compress(sub_path_timestamps, masks))]

        for version in versions[model_name]:
            masks = list(
                map(
                    lambda x: x.startswith(os.path.join(model_path, version) + "/"),
                    bucket_sub_paths,
                )
            )
            model_ts.append(max(itertools.compress(sub_path_timestamps, masks)))

        aux_timestamps.append(model_ts)

    timestamps = aux_timestamps  # type: List[List[datetime.datetime]]

    # model_names - a list with the names of the models (i.e. bert, gpt-2, etc) and they are unique
    # versions - a dictionary with the keys representing the model names and the values being lists of versions that each model has.
    #   For non-versioned model paths ModelVersion.NOT_PROVIDED, the list will be empty
    # model_paths - a list with the prefix of each model
    # sub_paths - a list of filepaths lists for each file of each model
    # timestamps - a list of timestamps lists representing the last edit time of each versioned model
    # bucket_names - names of the buckets

    return model_names, versions, model_paths, sub_paths, timestamps, bucket_names
