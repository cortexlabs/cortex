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
import os
import shutil
import threading as td
import time
from typing import Dict, List, Any, Tuple, Callable, Optional

from cortex_internal.lib.concurrency import ReadWriteLock
from cortex_internal.lib.exceptions import WithBreak
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.type import HandlerType

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


class ModelsHolder:
    """
    Class to hold models in memory and references for those on disk.
    Can limit the number of models in memory/on-disk based on an LRU policy - by default, it's disabled.
    """

    def __init__(
        self,
        handler_type: HandlerType,
        model_dir: str,
        temp_dir: str = "/tmp/cron",
        mem_cache_size: int = -1,
        disk_cache_size: int = -1,
        on_download_callback: Optional[
            Callable[[HandlerType, str, str, str, str, str, str, str], datetime.datetime]
        ] = None,
        on_load_callback: Optional[Callable[[str], Any]] = None,
        on_remove_callback: Optional[Callable[[List[str]], None]] = None,
    ):
        """
        Args:
            handler_type: The handler type. Can be PythonHandler, TensorFlowHandler or TensorFlowNeuronHandler.
            model_dir: Where models are saved on disk.
            temp_dir: Where models are temporary stored for validation.
            mem_cache_size: The size of the cache for in-memory models. For negative values, the cache is disabled.
            disk_cache_size: The size of the cache for on-disk models. For negative values, the cache is disabled.
            on_download_callback(<handler_type>, <bucket-name>, <model_name>, <model_version>, <model_path>, <temp_dir>, <model_dir>): Function to be called for downloading a model to disk. Returns the downloaded model's upstream timestamp, otherwise a negative number is returned.
            on_load_callback(<disk_model_path>, **kwargs): Function to be called when a model is loaded from disk. Returns the actual model. May throw exceptions if it doesn't work.
            on_remove_callback(<list of model IDs to remove>, **kwargs): Function to be called when the GC is called. E.g. for the TensorFlow type, the function would communicate with TFS to unload models.
        """
        self._handler_type = handler_type
        self._model_dir = model_dir
        self._temp_dir = temp_dir

        if mem_cache_size > 0 and disk_cache_size > 0 and mem_cache_size > disk_cache_size:
            raise RuntimeError(
                f"mem_cache_size ({mem_cache_size}) must be equal or smaller than disk_cache_size ({disk_cache_size})"
            )

        if mem_cache_size == 0 or disk_cache_size == 0:
            raise RuntimeError(
                "mem_cache_size or disk_cache_size can't be set to 0; must be negative to disable the cache or positive to have it enabled"
            )

        self._mem_cache_size = mem_cache_size
        self._disk_cache_size = disk_cache_size

        self._download_callback = on_download_callback
        self._load_callback = on_load_callback
        self._remove_callback = on_remove_callback

        self._models = {}  # maps the model ID to the model that's placed in memory
        self._timestamps = {}  # maps the model ID to the last access time of the model
        self._locks = {}  # maps the model ID to the underlying lock for each model

        self._create_lock = (
            td.RLock()
        )  # to ensure atomicity when 2 threads are trying to create locks for the same model ID that doesn't exist in self._locks
        self._global_lock = ReadWriteLock()

    def set_callback(self, ctype: str, callback: Callable) -> None:
        """
        Sets a callback.

        Args:
            ctype: "download", "load" or "remove" callback type - see the constructor to mark each one.
            callback: The actual callback.
        """
        if ctype == "download":
            self._download_callback = callback
        if ctype == "load":
            self._load_callback = callback
        if ctype == "remove":
            self._remove_callback = callback

    def global_acquire(self, mode: str, timeout: Optional[float] = None) -> None:
        """
        Acquire shared/exclusive (R/W) access over all models.

        Use "w" when wanting to acquire exclusive access for the GC (method garbage_collect), or "r" when wanting to grant shared access for any other method to be called (i.e. get_model_ids).

        Args:
            mode: "r" for read lock, "w" for write lock.
            timeout: How many seconds to wait to acquire the lock.
        """
        self._global_lock.acquire(mode, timeout)

    def global_release(self, mode: str) -> None:
        """
        Release shared/exclusive (R/W) access over all models.

        Args:
            mode: "r" for read lock, "w" for write lock.
        """
        self._global_lock.release(mode)

    def model_acquire(self, mode: str, model_name: str, model_version: str) -> None:
        """
        Acquire shared/exclusive (R/W) access for a specific model.

        Args:
            mode: "r" for read lock, "w" for write lock.
            model_name: The name of the model.
            model_version: The version of the model.

        When mode is "r", only the following methods can be called:
        * has_model
        * get_model

        When mode is "w", the methods available for "r" can be called plus the following ones:
        * load_model
        * download_model
        * remove_model
        """
        model_id = f"{model_name}-{model_version}"

        if not model_id in self._locks:
            lock = ReadWriteLock()
            self._create_lock.acquire()
            if model_id not in self._locks:
                self._locks[model_id] = lock
            self._create_lock.release()

        self._locks[model_id].acquire(mode)

    def model_release(self, mode: str, model_name: str, model_version: str) -> None:
        """
        Release shared/exclusive (R/W) access for a specific model.

        Args:
            mode: "r" for read lock, "w" for write lock.
            model_name: The name of the model.
            model_version: The version of the model.
        """
        model_id = f"{model_name}-{model_version}"
        self._locks[model_id].release(mode)

    def set_global_preference_policy(self, prefer: bool) -> bool:
        """
        Wrapper of cortex_internal.lib.concurrency.ReadWriteLock.set_preference_policy.
        """
        return self._global_lock.set_preference_policy(prefer)

    def get_model_names_by_tag_count(self, tag: str, count: int) -> Tuple[List[str], List[int]]:
        """
        Filter model names by the tag count based on the latest recently used model version.

        Locking is already done within the method.

        Args:
            tag: Tag as passed on in load_model method. If tag is not found, then the model is not considered.
            count: How many appearances a tag has to make for a given model name to be selected.

        Returns:
            List of model names that abide by the method's selection rule.
            List of timestamps representing the latest upstream timestamp that abide by the method's selection rule.
        """

        models = {}
        with LockedGlobalModelsGC(self, "r"):
            models_ids = self.get_model_ids()

        for model_id in models_ids:

            model_name, model_version = model_id.rsplit("-", maxsplit=1)
            with LockedModel(self, "r", model_name, model_version):
                if not self.has_model_id(model_id):
                    raise WithBreak

                tag_count = self._models[model_id]["metadata"]["consecutive_tag_count"][tag]
                ts = self._timestamps[model_id]

                if (
                    model_name in models and models[model_name]["timestamp"] < ts
                ) or model_name not in models:
                    models[model_name]["timestamp"] = ts
                    models[model_name]["count"] = tag_count

        filtered_model_names = []
        filtered_model_ts = []
        for model_name, v in models:
            if v["count"] >= count:
                filtered_model_names.append(model_name)
                filtered_model_ts.append(v["timestamp"])

        return filtered_model_names, filtered_model_ts

    def has_model(self, model_name: str, model_version: str) -> Tuple[str, int]:
        """
        Verifies if a model is loaded into memory / on disk.

        Args:
            model_name: The name of the model.
            model_version: The version of the model.

        Returns:
            "in-memory" and the upstream timestamp of the model when the model is loaded into memory. "in-memory" also implies "on-disk".
            "on-disk" and the upstream timestamp of the model when the model is saved to disk.
            "not-available" and 0 for the upstream timestamp when the model is not available.
        """
        model_id = f"{model_name}-{model_version}"
        if model_id in self._models:
            if self._models[model_id]["model"] is not None:
                return "in-memory", self._models[model_id]["upstream_timestamp"]
            else:
                return "on-disk", self._models[model_id]["upstream_timestamp"]
        return "not-available", 0

    def has_model_id(self, model_id: str) -> Tuple[str, int]:
        """
        Wrapper for has_model method.
        """
        model_name, model_version = model_id.rsplit("-", maxsplit=1)
        return self.has_model(model_name, model_version)

    def get_model(
        self, model_name: str, model_version: str, version_tag: str = ""
    ) -> Tuple[Any, int]:
        """
        Retrieves a model from memory.

        If the returned model is None, but the upstream timestamp is positive, then it means the model is present on disk.
        If the returned model is None and the upstream timestamp is 0, then the model is not present.
        If the returned model is not None, then the upstream timestamp will also be positive.

        Args:
            model_name: The name of the model.
            model_version: The version of the model.
            version_tag: The tag associated with the given model. If the tag is present, its count will be increased by one.

        Returns:
            The model and the model's upstream timestamp.
        """
        model_id = f"{model_name}-{model_version}"

        if model_id in self._models:
            self._timestamps[model_id] = time.time()

            if version_tag in self._models[model_id]["metadata"]["consecutive_tag_count"]:
                self._models[model_id]["metadata"]["consecutive_tag_count"][version_tag] += 1
            else:
                for tag in self._models[model_id]["metadata"]["consecutive_tag_count"]:
                    self._models[model_id]["metadata"]["consecutive_tag_count"][tag] = 0

            return self._models[model_id]["model"], self._models[model_id]["upstream_timestamp"]

        return None, 0

    def load_model(
        self,
        model_name: str,
        model_version: str,
        upstream_timestamp: int,
        tags: List[str] = [],
        kwargs: dict = {},
    ) -> None:
        """
        Loads a given model into memory.
        It is assumed the model already exists on disk. The model must be downloaded externally or with download_model method.

        Args:
            model_name: The name of the model.
            model_version: The version of the model.
            upstream_timestamp: When was this model last modified on the upstream source (S3 only).
            tags: List of tags to initialize the model with.
            kwargs: Extra arguments to pass into the loading callback.

        Raises:
            RuntimeError if a load callback isn't set. Can also raise exception if the load callback raises.
        """

        if self._load_callback:
            model_id = f"{model_name}-{model_version}"
            disk_path = os.path.join(self._model_dir, model_name, model_version)

            model = {
                "model": self._load_callback(disk_path, **kwargs),
                "disk_path": disk_path,
                "upstream_timestamp": upstream_timestamp,
                "metadata": {
                    "consecutive_tag_count": {},
                },
            }
            if len(tags) > 0:
                for tag in tags:
                    model["metadata"]["consecutive_tag_count"][tag] = 0

            self._models[model_id] = model
        else:
            raise RuntimeError(
                "a load callback must be provided; use set_callback to set a callback"
            )

    def download_model(
        self,
        bucket: str,
        model_name: str,
        model_version: str,
        model_path: str,
    ) -> datetime.datetime:
        """
        Download a model to disk. To be called before load_model method is called.

        To be used when the caching is enabled.
        It is assumed that when caching is disabled, an external mechanism is responsible for downloading/removing models to/from disk.

        Args:
            bucket: The upstream model's bucket name.
            model_name: The name of the model.
            model_version: The version of the model.
            model_path: Path to the model as discovered in models:dir or specified in models:paths.

        Returns:
            Returns the downloaded model's upstream timestamp, otherwise None is returned if it fails.

        Raises:
            Exceptions if the download callback raises any.
        """
        if self._download_callback:
            return self._download_callback(
                self._handler_type,
                bucket,
                model_name,
                model_version,
                model_path,
                self._temp_dir,
                self._model_dir,
            )
        raise RuntimeError(
            "a download callback must be provided; use set_callback to set a callback"
        )

    def unload_model(self, model_name: str, model_version: str, kwargs: dict = {}) -> None:
        """
        Unloads a model from memory. If applicable, it gets called before remove_model/remove_model_by_id.

        Args:
            model_name: The name of the model.
            model_version: The version of the model.
            kwargs: Passable arguments to the remove callback.

        Raises:
            Exceptions if the remove callback raises any.
        """

        if self._remove_callback:
            model_id = f"{model_name}-{model_version}"
            self._remove_callback([model_id], **kwargs)

    def remove_model(self, model_name: str, model_version: str) -> None:
        """
        Removes a model from memory and disk if it exists.
        """
        model_id = f"{model_name}-{model_version}"
        self.remove_model_by_id(model_id, True, True)

    def remove_model_by_id(
        self, model_id: str, mem: bool, disk: bool, del_reference: bool = False
    ) -> None:
        """
        Remove a model from this object and/or from disk.

        Args:
            model_id: The model ID to remove.
            mem: Whether to remove the model from memory or not.
            disk: Whether to remove the model from disk or not.
            del_reference: Whether to remove the model reference or not. Don't touch this unless you know what you do.
        """
        if model_id not in self._models:
            return None

        if mem:
            # remove model from memory (but keeps it on disk)
            self._models[model_id]["model"] = None

        if disk:
            disk_path = self._models[model_id]["disk_path"]
            shutil.rmtree(disk_path)

        if disk or del_reference:
            del self._models[model_id]
            del self._timestamps[model_id]

    def garbage_collect(
        self, exclude_disk_model_ids: List[str] = [], dry_run: bool = False
    ) -> Tuple[bool, List[str], List[str]]:
        """
        Removes stale in-memory and on-disk models based on LRU policy.
        Also calls the "remove" callback before removing the models from this object. The callback must not raise any exceptions.

        Must be called with a write lock unless dry_run is set to true.

        Args:
            exclude_disk_model_ids: Model IDs to exclude from removing from disk. Necessary for locally-provided models.
            dry_run: Just test if there are any models to remove. If set to true, this method can then be called with a read lock.

        Returns:
            A 3-element tuple. First element tells whether models had to be collected. The 2nd and 3rd elements contain the model IDs that were removed from memory and disk respectively.
        """
        collected = False
        if self._mem_cache_size <= 0 or self._disk_cache_size <= 0:
            return collected

        stale_mem_model_ids = self._lru_model_ids(self._mem_cache_size, filter_in_mem=True)
        stale_disk_model_ids = self._lru_model_ids(
            self._disk_cache_size - len(exclude_disk_model_ids), filter_in_mem=False
        )

        if self._remove_callback and not dry_run:
            self._remove_callback(stale_mem_model_ids)

        # don't delete excluded model IDs from disk
        stale_disk_model_ids = list(set(stale_disk_model_ids) - set(exclude_disk_model_ids))
        stale_disk_model_ids = stale_disk_model_ids[
            len(stale_disk_model_ids) - self._disk_cache_size :
        ]

        if not dry_run:
            logger.info(
                f"unloading models {stale_mem_model_ids} from memory using the garbage collector"
            )
            logger.info(
                f"unloading models {stale_disk_model_ids} from disk using the garbage collector"
            )
            for model_id in stale_mem_model_ids:
                self.remove_model_by_id(model_id, mem=True, disk=False)
            for model_id in stale_disk_model_ids:
                self.remove_model_by_id(model_id, mem=False, disk=True)

        if len(stale_mem_model_ids) > 0 or len(stale_disk_model_ids) > 0:
            collected = True

        return collected, stale_mem_model_ids, stale_disk_model_ids

    def get_model_ids(self) -> List[str]:
        """
        Gets a list of all loaded model IDs (in memory or on disk).
        """
        return list(self._models.keys())

    def _lru_model_ids(self, threshold: int, filter_in_mem: bool) -> List[str]:
        """
        Sort model ids by last access and get the model ids with ranks below the specified threshold.

        Args:
            threshold: The memory cache size or the disk cache size.
            filter_in_mem: In the counting process, set whether to only look at models loaded in memory or not. True for only looking at models loaded in memory and on disk.

        Returns:
            A list of stale model IDs.
        """
        copied_timestamps = self._timestamps.copy()
        timestamps = {
            k: v
            for k, v in sorted(copied_timestamps.items(), key=lambda item: item[1], reverse=True)
        }
        model_ids = []
        for counter, model_id in enumerate(timestamps):
            # skip models if they are not loaded in memory but on disk
            if filter_in_mem and self._models[model_id]["model"] is None:
                continue
            if counter >= threshold:
                model_ids.append(model_id)

        return model_ids


def ids_to_models(model_ids: List[str]) -> Dict[str, List[str]]:
    """
    Convert model IDs (MODEL_NAME-MODEL_VERSION) to a dictionary with its keys being
    the model names and its values being lists of the associated versions for each given model name.
    """

    models = {}
    for model_id in model_ids:
        model_name, model_version = model_id.rsplit("-", maxsplit=1)
        if model_name not in models:
            models[model_name] = [model_version]
        else:
            models[model_name].append(model_version)
    return models


class LockedGlobalModelsGC:
    """
    Applies global exclusive lock (R/W) on the models holder.

    For running the GC for all loaded models (or present on disk).
    This is the locking implementation for the stop-the-world GC.

    The context manager can be exited by raising cortex_internal.lib.exceptions.WithBreak.
    """

    def __init__(
        self,
        models: ModelsHolder,
        mode: str = "w",
        prefer: str = "r",
        timeout: Optional[float] = None,
    ):
        self._models = models
        self._mode = mode
        self._timeout = timeout

    def __enter__(self):
        self.acquired = True
        try:
            self._models.global_acquire(self._mode, self._timeout)
        except TimeoutError:
            self.acquired = False
        return self

    def __exit__(self, exc_type, exc_value, traceback) -> bool:
        self._models.global_release(self._mode)

        if exc_value is not None and exc_type is not WithBreak:
            return False
        return True


class LockedModel:
    """
    For granting shared/exclusive (R/W) access to a model resource (model name + model version).
    Also applies global read lock on the models holder.

    The context manager can be exited by raising cortex_internal.lib.exceptions.WithBreak.
    """

    def __init__(
        self,
        models: ModelsHolder,
        mode: str,
        model_name: str = "",
        model_version: str = "",
        model_id: str = "",
    ):
        """
        mode can be "r" for read or "w" for write.
        """
        self._models = models
        self._mode = mode
        if model_id != "":
            self._model_name, self._model_version = model_id.rsplit("-", maxsplit=1)
        else:
            self._model_name = model_name
            self._model_version = model_version

    def __enter__(self):
        self._models.global_acquire("r")
        self._models.model_acquire(self._mode, self._model_name, self._model_version)
        return self

    def __exit__(self, exc_type, exc_value, traceback) -> bool:
        self._models.model_release(self._mode, self._model_name, self._model_version)
        self._models.global_release("r")

        if exc_value is not None and exc_type is not WithBreak:
            return False
        return True
