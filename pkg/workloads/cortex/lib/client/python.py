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
import datetime
import threading as td
import multiprocessing as mp
from typing import Any, Optional, Callable

from cortex.lib.log import cx_logger as logger
from cortex.lib.exceptions import UserRuntimeException, CortexException, UserException, WithBreak
from cortex.lib.model import (
    ModelsHolder,
    LockedModel,
    ModelsTree,
    LockedModelsTree,
    CuratedModelResources,
    find_ondisk_model_info,
    find_ondisk_models_with_lock,
)
from cortex.lib.concurrency import LockedFile
from cortex import consts


class PythonClient:
    def __init__(
        self,
        api_spec: dict,
        models: ModelsHolder,
        model_dir: str,
        models_tree: Optional[ModelsTree],
        lock_dir: Optional[str] = "/run/cron",
        load_model_fn: Optional[Callable[[str], Any]] = None,
    ):
        """
        Setup Python model client.

        Args:
            api_spec: API configuration.

            models: Holding all models into memory.
            model_dir: Where the models are saved on disk.

            models_tree: A tree of the available models from upstream.
            lock_dir: Where the resource locks are found. Only when processes_per_replica > 0 and caching disabled.
            load_model_fn: Function to load model into memory.
        """

        self._api_spec = api_spec
        self._models = models
        self._models_tree = models_tree
        self._model_dir = model_dir
        self._lock_dir = lock_dir

        self._spec_models = CuratedModelResources(api_spec["curated_model_resources"])

        if (
            self._api_spec["predictor"]["models"]
            and self._api_spec["predictor"]["models"]["dir"] is not None
        ):
            self._models_dir = True
        else:
            self._models_dir = False
            self._spec_model_names = self._spec_models.get_field("name")

        # for when local models are used
        self._spec_local_model_names = self._spec_models.get_local_model_names()
        self._local_model_ts = int(datetime.datetime.now(datetime.timezone.utc).timestamp())

        self._multiple_processes = self._api_spec["predictor"]["processes_per_replica"] > 1
        self._caching_enabled = self._is_model_caching_enabled()

        if callable(load_model_fn):
            self._models.set_callback("load", load_model_fn)

    def set_load_method(self, load_model_fn: Callable[[str], Any]) -> None:
        self._models.set_callback("load", load_model_fn)

    def get_model(self, model_name: Optional[str] = None, model_version: str = "latest") -> Any:
        """
        Retrieve a model for inference.

        Args:
            model_name (optional): Name of the model to retrieve (when multiple models are deployed in an API).
                When predictor.models.paths is specified, model_name should be the name of one of the models listed in the API config.
                When predictor.models.dir is specified, model_name should be the name of a top-level directory in the models dir.
            model_version (string, optional): Version of the model to retrieve. Can be omitted or set to "latest" to select the highest version.

        Returns:
            The model as loaded by the load_model() method.
        """

        if model_version != "latest" and not model_version.isnumeric():
            raise UserRuntimeException(
                "model_version must be either a parse-able numeric value or 'latest'"
            )

        # when predictor:model_path or predictor:models:paths is specified
        if not self._models_dir:

            # when predictor:model_path is provided
            if consts.SINGLE_MODEL_NAME in self._spec_model_names:
                model_name = consts.SINGLE_MODEL_NAME
                model = self._get_model(model_name, model_version)
                if model is None:
                    raise UserRuntimeException(
                        f"model {model_name} of version {model_version} wasn't found"
                    )
                return model

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
        if self._models_dir:
            if model_name is None:
                raise UserRuntimeException("model_name was not specified")
            if not self._caching_enabled:
                available_models = find_ondisk_models_with_lock(self._lock_dir)
                if model_name not in available_models:
                    raise UserRuntimeException(
                        f"'{model_name}' model wasn't found in the list of available models"
                    )

        model = self._get_model(model_name, model_version)
        if model is None:
            raise UserRuntimeException(
                f"model {model_name} of version {model_version} wasn't found"
            )
        return model

    def _get_model(self, model_name: str, model_version: str) -> Any:
        """
        Checks if versioned model is on disk, then checks if model is in memory,
        and if not, it loads it into memory, and returns the model.

        Args:
            model_name: Name of the model, as it's specified in predictor:models:paths or in the other case as they are named on disk.
            model_version: Version of the model, as it's found on disk. Can also infer the version number from the "latest" tag.

        Exceptions:
            RuntimeError: if another thread tried to load the model at the very same time.

        Returns:
            The model as returned by self._load_model method.
            None if the model wasn't found or if it didn't pass the validation.
        """

        model = None
        tag = ""
        if model_version == "latest":
            tag = model_version

        if not self._caching_enabled:
            # determine model version
            if tag == "latest":
                model_version = self._get_latest_model_version_from_disk(model_name)
            model_id = model_name + "-" + model_version

            # grab shared access to versioned model
            resource = os.path.join(self._lock_dir, model_id + ".txt")
            with LockedFile(resource, "r", reader_lock=True) as f:

                # check model status
                file_status = f.read()
                if file_status == "" or file_status == "not-available":
                    raise WithBreak

                current_upstream_ts = int(file_status.split(" ")[1])
                update_model = False

                # grab shared access to models holder and retrieve model
                with LockedModel(self._models, "r", model_name, model_version):
                    status, local_ts = self._models.has_model(model_name, model_version)
                    if status == "not-available" or (
                        status == "in-memory" and local_ts != current_upstream_ts
                    ):
                        update_model = True
                        raise WithBreak
                    model, _ = self._models.get_model(model_name, model_version, tag)

                # load model into memory and retrieve it
                if update_model:
                    with LockedModel(self._models, "w", model_name, model_version):
                        status, _ = self._models.has_model(model_name, model_version)
                        if status == "not-available" or (
                            status == "in-memory" and local_ts != current_upstream_ts
                        ):
                            if status == "not-available":
                                logger().info(
                                    f"loading model {model_name} of version {model_version} (thread {td.get_ident()})"
                                )
                            else:
                                logger().info(
                                    f"reloading model {model_name} of version {model_version} (thread {td.get_ident()})"
                                )
                            try:
                                self._models.load_model(
                                    model_name,
                                    model_version,
                                    current_upstream_ts,
                                    [tag],
                                )
                            except Exception as e:
                                raise UserRuntimeException(
                                    f"failed (re-)loading model {model_name} of version {model_version} (thread {td.get_ident()})",
                                    str(e),
                                )
                        model, _ = self._models.get_model(model_name, model_version, tag)

        if not self._multiple_processes and self._caching_enabled:
            # determine model version
            try:
                if tag == "latest":
                    model_version = self._get_latest_model_version_from_tree(
                        model_name, self._models_tree.model_info(model_name)
                    )
            except ValueError:
                # if model_name hasn't been found
                raise UserRuntimeException(
                    f"'{model_name}' model of tag latest wasn't found in the list of available models"
                )

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
                current_upstream_ts = int(upstream_model["timestamp"].timestamp())

            if not available_model:
                return None

            # grab shared access to models holder and retrieve model
            update_model = False
            with LockedModel(self._models, "r", model_name, model_version):
                status, local_ts = self._models.has_model(model_name, model_version)
                if status in ["not-available", "on-disk"] or (
                    status != "not-available"
                    and local_ts != current_upstream_ts
                    and not (status == "in-memory" and model_name in self._spec_local_model_names)
                ):
                    update_model = True
                    raise WithBreak
                model, _ = self._models.get_model(model_name, model_version, tag)

            # download, load into memory the model and retrieve it
            if update_model:
                # grab exclusive access to models holder
                with LockedModel(self._models, "w", model_name, model_version):

                    # check model status
                    status, local_ts = self._models.has_model(model_name, model_version)

                    # refresh disk model
                    if model_name not in self._spec_local_model_names and (
                        status == "not-available"
                        or (status in ["on-disk", "in-memory"] and local_ts != current_upstream_ts)
                    ):
                        if status == "not-available":
                            logger().info(
                                f"model {model_name} of version {model_version} not found locally; continuing with the download..."
                            )
                        elif status == "on-disk":
                            logger().info(
                                f"found newer model {model_name} of vesion {model_version} on the S3 upstream than the one on the disk"
                            )
                        else:
                            logger().info(
                                f"found newer model {model_name} of vesion {model_version} on the S3 upstream than the one loaded into memory"
                            )

                        # remove model from disk and memory
                        if status == "on-disk":
                            logger().info(
                                f"removing model from disk for model {model_name} of version {model_version}"
                            )
                            self._models.remove_model(model_name, model_version)
                        if status == "in-memory":
                            logger().info(
                                f"removing model from disk and memory for model {model_name} of version {model_version}"
                            )
                            self._models.remove_model(model_name, model_version)

                        # download model
                        logger().info(
                            f"downloading model {model_name} of version {model_version} from the S3 upstream"
                        )
                        date = self._models.download_model(
                            upstream_model["bucket"],
                            model_name,
                            model_version,
                            upstream_model["path"],
                        )
                        if not date:
                            raise WithBreak
                        current_upstream_ts = date.timestamp()

                    # give the local model a timestamp initialized at start time
                    if model_name in self._spec_local_model_names:
                        current_upstream_ts = self._local_model_ts

                    # load model
                    try:
                        logger().info(
                            f"loading model {model_name} of version {model_version} into memory"
                        )
                        self._models.load_model(
                            model_name,
                            model_version,
                            current_upstream_ts,
                            [tag],
                        )
                    except Exception as e:
                        raise UserRuntimeException(
                            f"failed (re-)loading model {model_name} of version {model_version} (thread {td.get_ident()})",
                            str(e),
                        )

                    # retrieve model
                    model, _ = self._models.get_model(model_name, model_version, tag)

        return model

    def _get_latest_model_version_from_disk(self, model_name: str) -> str:
        """
        Get the highest version for a specific model name.
        Must only be used when processes_per_replica > 0 and caching disabled.
        """
        versions, timestamps = find_ondisk_model_info(self._lock_dir, model_name)
        if len(versions) == 0:
            raise UserRuntimeException(
                "'{}' model's versions have been removed; add at least a version to the model to resume operations".format(
                    model_name
                )
            )
        return str(max(map(lambda x: int(x), versions)))

    def _get_latest_model_version_from_tree(self, model_name: str, model_info: dict) -> str:
        """
        Get the highest version for a specific model name.
        Must only be used when processes_per_replica = 1 and caching is enabled.
        """
        versions, timestamps = model_info["versions"], model_info["timestamps"]
        return str(max(map(lambda x: int(x), versions)))

    def _is_model_caching_enabled(self) -> bool:
        """
        Checks if model caching is enabled (models:cache_size and models:disk_cache_size).
        """
        return (
            self._api_spec["predictor"]["models"]
            and self._api_spec["predictor"]["models"]["cache_size"] is not None
            and self._api_spec["predictor"]["models"]["disk_cache_size"] is not None
        )

    @property
    def metadata(self) -> dict:
        """
        The returned dictionary will be like in the following example:
        {
            ...
            "yolov3": {
                "versions": [
                    "2",
                    "1"
                ],
                "timestamps": [
                    1601668127,
                    1601668127
                ]
            }
            ...
        }
        """
        if not self._caching_enabled:
            return find_ondisk_models_with_lock(self._lock_dir, include_timestamps=True)
        else:
            models_info = self._models_tree.get_all_models_info()
            for model_name in models_info.keys():
                del models_info[model_name]["bucket"]
                del models_info[model_name]["model_paths"]
            return models_info

    @property
    def caching(self) -> bool:
        return self._caching_enabled
