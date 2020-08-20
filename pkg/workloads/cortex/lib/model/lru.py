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

# Implementation of memory/disk-based LRU class
# ...

from typing import List, Any, Tuple, Callable, AbstractSet
from cortex.lib.exceptions import WithBreak

import os
import time
import shutil


class ModelsHolder:
    """
    Class to hold models in memory and references for those on disk.
    Can limit the number of models in memory/on-disk based on an LRU policy - by default, it's disabled.
    """

    def __init__(
        self,
        mem_cache_size: int = -1,
        disk_cache_size: int = -1,
        temp_dir: str = "/tmp/models",
        on_download_callback: Callable[[str, str, str, List[str]], int] = None,
        on_load_callback: Callable[[str], Any] = None,
        on_remove_callback: Callable[[List[str]], None] = None,
    ):
        """
        Args:
            mem_cache_size: The size of the cache for in-memory models. For negative values, the cache is disabled.
            disk_cache_size: The size of the cache for on-disk models. For negative values, the cache is disabled.
            on_download_callback(model_name, model_version, model_path, sub_paths): Function to be called for downloading a model to disk. Returns the downloaded model's upstream timestamp, otherwise a negative number is returned.
            on_load_callback(disk_model_path): Function to be called when a model is loaded from disk. Returns the actual model. May throw exceptions if it doesn't work.
            on_remove_callback(list of model IDs to remove): Function to be called when the GC is called. E.g. for the TensorFlow Predictor, the function would communicate with TFS to unload models.  
        """
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

        self._models = {}
        self._timestamps = {}
        self._locks = {}

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

    #################

    def global_acquire(self, mode: str) -> None:
        """
        Acquire R/W access over all models.

        The distinction between "r" and "w", is that for "w", any method of this class can be called, including the garbage_collect method.

        Args:
            mode: "r" for read lock, "w" for write lock.
        """
        self._global_lock.acquire(mode)

    def global_release(self, mode: str) -> None:
        """
        Release R/W access over all models.

        Args:
            mode: "r" for read lock, "w" for write lock.
        """
        self._global_lock.release(mode)

    def model_acquire(self, mode: str, model_name: str, model_version: str) -> None:
        """
        Acquire R/W access for a specific model.

        Args:
            mode: "r" for read lock, "w" for write lock.
            model_name: The name of the model.
            model_version: The version of the model.

        When mode is "r", only the following methods should be called:
        * has_model
        * get_model

        When mode is "w", the methods available for "r" can be called plus the following ones:
        * load_model
        * remove_model
        """
        model_id = f"{model_name}-{model_version}"

        if not model_id in self._locks:
            lock = ReadWriteLock()
            lock_id = id(lock)
            self._locks[model_id] = lock
            self._locks[model_id].acquire(mode)
            if id(self._locks[model_id]) is not lock_id:
                self._locks[model_id].release(mode)
                raise RuntimeError("caught lock generated by another thread; retry")
        else:
            self._locks[model_id].acquire(mode)

    def model_release(self, mode: str, model_name: str, model_version: str) -> None:
        """
        Release R/W access for a specific model.

        Args:
            mode: "r" for read lock, "w" for write lock.
            model_name: The name of the model.
            model_version: The version of the model.
        """
        model_id = f"{model_name}-{model_version}"
        self._locks[model_id].release(mode)

    #################

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
        if model_id in self._models and self._models[model_id]["model"] is not None:
            return True, self._models[model_id]["upstream_timestamp"]
        if model_id in self._models:
            if self._models[model_id]["model"] is not None:
                return "in-memory", self._models[model_id]["upstream_timestamp"]
            else:
                return "on-disk", self._models[model_id]["upstream_timestamp"]
        return "not-available", 0

    def get_model(self, model_name: str, model_version: str) -> Tuple[Any, int]:
        """
        Retrieves a model from memory.

        If the returned model is None, but the upstream timestamp is positive, then it means the model is present on disk.
        If the returned model is None and the upstream timestamp is 0, then the model is not present.
        If the returned model is not None, then the upstream timestamp will also be positive.

        Args:
            model_name: The name of the model.
            model_version: The version of the model.

        Returns:
            The model and the model's upstream timestamp.
        """
        model_id = f"{model_name}-{model_version}"
        if model_id in self._models:
            self._timestamps[model_id] = time.time()
            return self._models[model_id]["model"], self._models[model_id]["upstream_timestamp"]
        return None, 0

    def load_model(
        self, model_name: str, model_version: str, disk_path: str, upstream_timestamp: int,
    ) -> None:
        """
        Loads a given model into memory.
        It is assumed the model already exists on disk. The model must be downloaded externally or with download_model method.

        Args:
            model_name: The name of the model.
            model_version: The version of the model.
            disk_path: Where the model is found.
            upstream_timestamp: When was this model last modified on the upstream source (e.g. S3).

        Raises:
            RuntimeError if a load callback isn't set.
        """

        if self._load_callback:
            model_id = f"{model_name}-{model_version}"
            self._models[model_id] = {
                "model": self._load_callback(disk_path),
                "disk_path": disk_path,
                "upstream_timestamp": upstream_timestamp,
            }
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
        sub_paths: List[str],
    ) -> int:
        """
        Download a model to disk. To be called before load_model method is called.

        To be used when the caching is enabled.
        It is assumed that when caching is disabled, an external mechanism is responsible for downloading/removing models to/from disk.

        Args:
            bucket: The upstream model's S3 bucket name.
            model_name: The name of the model.
            model_version: The version of the model.
            model_path: Path to the model as discovered in models:dir or specified in models:paths.
            sub_paths: A list of filepaths for each file of the model.

        Returns:
            Returns the downloaded model's upstream timestamp, otherwise a negative number is returned.
        """
        if self._download_callback:
            return self._download_callback(bucket, model_name, model_version, model_path, sub_paths)
        raise RuntimeError(
            "a download callback must be provided; use set_callback to set a callback"
        )

    def remove_model(self, model_name: str, model_version: str,) -> None:
        """
        Removes a model from memory if it exists.

        Note: not required anywhere yet.
        """
        model_id = f"{model_name}-{model_version}"
        if model_id in self._models:
            del self._models[model_id]
        if model_id in self._timestamps:
            del self._timestamps[model_id]

    #################

    def garbage_collect(self) -> bool:
        """
        Removes stale in-memory and on-disk models based on LRU policy.
        Also calls the "remove" callback before removing the models from this object.

        Returns:
            True when models had to be collected. False otherwise.
        """
        collected = False
        if self._mem_cache_size <= 0 or self._disk_cache_size <= 0:
            return collected

        stale_mem_model_ids = self._lru_model_ids(self._mem_cache_size)
        stale_disk_model_ids = self._lru_model_ids(self._disk_cache_size)

        if self._callback:
            self._callback(stale_mem_model_ids)

        for model_id in stale_mem_model_ids:
            self._remove_model(model_id, mem=True, disk=False)
        for model_id in stale_disk_model_ids:
            self._remove_model(model_id, mem=False, disk=True)

        if len(stale_mem_model_ids) > 0 or len(stale_disk_model_ids) > 0:
            collected = True

        return collected

    def _lru_model_ids(self, threshold: int) -> List[str]:
        """
        Looking at the LRU, this method gets the model IDs which find themselves at higher indexes than the threshold.

        Args:
            threshold: The memory cache size or the disk cache size.

        Returns:
            A list of stale model IDs.
        """
        self._timestamps = {
            k: v
            for k, v in sorted(self._timestamps.items(), key=lambda item: item[1], reverse=True)
        }
        model_ids = []
        for counter, model_id in enumerate(self._timestamps):
            counter += 1
            if counter > threshold:
                model_ids.append(model_id)

        return model_ids

    def _remove_model(self, model_id: str, mem: bool, disk: bool) -> None:
        """
        Remove a model from this object and/or from disk.

        Args:
            model_id: The model ID to remove.
            mem: Whether to remove the model from memory or not.
            disk: Whether to remove the model from disk or not.
        """
        if mem:
            self._models[model_id]["model"] = None
        if disk:
            disk_path = self._models[model_id]["disk_path"]
            shutil.rmtree(disk_path)
            model_top_dir = os.path.dirname(disk_path)
            if len(os.listdir(model_top_dir)) == 0:
                shutil.rmtree(model_top_dir)

            del self._models[model_id]
            del self._timestamps[model_id]

    #################


class LockedGlobalModelsGC:
    """
    Applies global write lock on the models tree.

    For running the GC for all loaded models (or present on disk).
    This is the locking implementation for the stop-the-world GC.

    The context manager can be exited by raising cortex.lib.exceptions.WithBreak.
    """

    def __init__(self, models: ModelsHolder):
        self._models = models

    def __enter__(self):
        self._models.global_acquire("w")
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self._models.global_release("w")

        if exc_value is not None and exc_type is not WithBreak:
            raise exc_type(exc_value).with_traceback(traceback)


class LockedModel:
    """
    For granting R/W access to a model resource (model name + model version).
    Also applies global read lock on the models tree.

    The context manager can be exited by raising cortex.lib.exceptions.WithBreak.
    """

    def __init__(self, models: ModelsHolder, mode: str, model_name: str, model_version: str):
        self._models = models
        self._mode = mode
        self._model_name = model_name
        self._model_version = model_version

    def __enter__(self):
        self._models.global_acquire("r")
        self._models.model_acquire(self._mode, self._model_name, self._model_version)
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self._models.model_release(self._mode, self._model_name, self._model_version)
        self._models.global_release("r")

        if exc_value is not None and exc_type is not WithBreak:
            raise exc_type(exc_value).with_traceback(traceback)
