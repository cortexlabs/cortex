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

            load_model_fn: Function to load model into memory.
            models: Holding all models into memory.
            model_dir: Where the models are saved on disk.

            models_tree: A tree of the available models from upstream.
            lock_dir: Where the resource locks are found. Only when processes_per_replica > 0 and caching disabled.
            load_model_fn: Model loader function.
        """

        self._api_spec = api_spec
        self._models = models
        self._models_tree = models_tree
        self._model_dir = model_dir
        self._lock_dir = lock_dir

        self._spec_models = CuratedModelResources(api_spec["curated_model_resources"])

        if self._api_spec["predictor"]["models"]["dir"] is not None:
            self._models_dir = True
        else:
            self._models_dir = False
            self._spec_model_names = self._spec_models.get_field("name")

        self._multiple_processes = self._api_spec["predictor"]["processes_per_replica"] > 1
        self._caching_enabled = self._is_model_caching_enabled()

        if callable(load_model_fn):
            self._models.set_callback("load", load_model_fn)

    def set_load_method(self, load_model_fn: Callable[[str], Any]) -> None:
        self._models.set_callback("load", load_model_fn)

    def get_model(self, model_name: Optional[str] = None, model_version: str = "highest") -> Any:
        """
        Retrieve model for inference.

        Args:
            model_name: Model to use when multiple models are deployed in a single API.
            model_version: Model version to use. Can also be "highest" for picking the highest version or "latest" for picking the most recent version.

        Returns:
            The model as loaded by load_model method.
        """

        if model_version not in ["highest", "latest"] and not model_version.isnumeric():
            raise UserRuntimeException(
                "model_version must be either a parse-able numeric value or 'highest' or 'latest'"
            )

        # when predictor:model_path or predictor:models:paths is specified
        if self._models_dir is None:

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
            model_version: Version of the model, as it's found on disk. Can also infer the version number from "latest" and "highest" tags.

        Exceptions:
            RuntimeError: if another thread tried to load the model at the very same time.

        Returns:
            The model as returned by self._load_model method.
            None if the model wasn't found or if it didn't pass the validation.
        """

        model = None
        tag = ""
        tags = []
        if model_version in ["latest", "highest"]:
            tag = model_version
            tags = ["latest", "highest"]

        if not self._caching_enabled:
            # determine model version
            if tag != "":
                model_version = self._get_model_version_from_disk(model_name, tag)
            model_id = model_name + "-" + model_version

            # grab shared access to versioned model
            resource = os.path.join(self._lock_dir, model_id + ".txt")
            with LockedFile(resource, "r", reader_lock=True) as f:

                # check model status
                file_status = f.read()
                if file_status == "" or file_status == "not available":
                    raise WithBreak

                current_upstream_ts = int(file_status.split(" ")[1])
                update_model = False

                # grab shared access to models holder and retrieve model
                with LockedModel(self._models, "r", model_name, model_version):
                    status, upstream_ts = self._models.has_model(model_name, model_version)
                    if status == "not-available" or (
                        status == "in-memory" and upstream_ts < current_upstream_ts
                    ):
                        update_model = True
                        raise WithBreak
                    model, _ = self._models.get_model(model_name, model_version, tag)

                # load model into memory and retrieve it
                if update_model:
                    with LockedModel(self._models, "w", model_name, model_version):
                        status, _ = self._models.has_model(model_name, model_version)
                        if status == "not-available" or (
                            status == "in-memory" and upstream_ts < current_upstream_ts
                        ):
                            if status == "not-available":
                                logger().info(
                                    f"loading model {model_name} of version {model_version} (process {mp.current_process().pid}, thread {td.get_ident()})"
                                )
                            else:
                                logger().info(
                                    f"reloading model {model_name} of version {model_version} (process {mp.current_process().pid}, thread {td.get_ident()})"
                                )
                            try:
                                self._models.load_model(
                                    model_name,
                                    model_version,
                                    current_upstream_ts,
                                    tags,
                                )
                            except Exception as e:
                                raise UserRuntimeException(
                                    f"failed (re-)loading model {model_name} of version {model_version} (process {mp.current_process().pid}, thread {td.get_ident()})",
                                    str(e),
                                )
                        model, _ = self._models.get_model(model_name, model_version, tag)

        if not self._multiple_processes and self._caching_enabled:
            # determine model version
            try:
                if tag != "":
                    model_version = self._get_model_version_from_tree(
                        model_name, tag, self._models_tree.model_info(model_name)
                    )
            except ValueError:
                # if model_name hasn't been found
                raise UserRuntimeException(
                    f"'{model_name}' model of tag {model_version} wasn't found in the list of available models"
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
                current_upstream_ts = upstream_model["timestamp"]

            if not available_model:
                return None

            # grab shared access to models holder and retrieve model
            update_model = False
            with LockedModel(self._models, "r", model_name, model_version):
                status, upstream_ts = self._models.has_model(model_name, model_version)
                if status in ["not-available", "on-disk"] or (
                    status != "not-available" and upstream_ts < current_upstream_ts
                ):
                    update_model = True
                    raise WithBreak
                model, _ = self._models.get_model(model_name, model_version, tag)

            # download, load into memory the model and retrieve it
            if update_model:
                # grab exclusive access to models holder
                with LockedModel(self._models, "w", model_name, model_version):

                    # check model status
                    status, upstream_ts = self._models.has_model(model_name, model_version)

                    # refresh disk model
                    if status == "not-available" or (
                        status in ["on-disk", "in-memory"] and upstream_ts < current_upstream_ts
                    ):
                        # remove model from disk and references
                        if status in ["on-disk", "in-memory"]:
                            logger().info(
                                f"removing model references from memory and from disk for {model_name} {model_version}"
                            )
                            self._models.remove_model(model_name, model_version)

                        # download model
                        date = self._models.download_model(
                            upstream_model["bucket"],
                            model_name,
                            model_version,
                            upstream_model["path"],
                        )
                        if not date:
                            raise WithBreak
                        current_upstream_ts = date.timestamp()

                    # load model
                    try:
                        self._models.load_model(
                            model_name,
                            model_version,
                            current_upstream_ts,
                            tags,
                        )
                    except Exception:
                        raise WithBreak

                    # retrieve model
                    model, _ = self._models.get_model(model_name, model_version, tag)

        return model

    def _get_model_version_from_disk(self, model_name: str, tag: str) -> str:
        """
        Get the version for a specific model name based on the version tag - either "latest" or "highest".
        Must only be used when processes_per_replica > 0 and caching disabled.
        """

        if tag not in ["latest", "highest"]:
            raise ValueError("invalid tag; must be either 'latest' or 'highest'")

        versions, timestamps = find_ondisk_model_info(self._lock_dir, model_name)
        if len(versions) == 0:
            raise UserRuntimeException(
                "'{}' model's versions have been removed; add at least a version to the model to resume operations".format(
                    model_name
                )
            )

        if tag == "latest":
            index = timestamps.index(max(timestamps))
            return versions[index]
        else:
            return str(max(map(lambda x: int(x), versions)))

    def _get_model_version_from_tree(self, model_name: str, tag: str, model_info: dict) -> str:
        """
        Get the version for a specific model name based on the version tag - either "latest" or "highest".
        Must only be used when processes_per_replica = 1 and caching is enabled.
        """

        if tag not in ["latest", "highest"]:
            raise ValueError("invalid tag; must be either 'latest' or 'highest'")

        versions, timestamps = model_info["versions"], model_info["timestamps"]
        if tag == "latest":
            index = timestamps.index(max(timestamps))
            return versions[index]
        else:
            return str(max(map(lambda x: int(x), versions)))

    def _is_model_caching_enabled(self) -> bool:
        """
        Checks if model caching is enabled (models:cache_size and models:disk_cache_size).
        """
        return (
            self._api_spec["predictor"]["models"]["cache_size"] is not None
            and self._api_spec["predictor"]["models"]["disk_cache_size"] is not None
        )

    # TODO retrieve sessions properly for cortex get
    @property
    def sessions(self):
        return None

    # TODO retrieve input_signatures properly for cortex get
    @property
    def input_signatures(self):
        return None
