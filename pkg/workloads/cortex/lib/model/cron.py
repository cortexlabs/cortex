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

from typing import Dict, List, Tuple, Any, Union, Callable

from cortex.lib import util
from cortex.lib.log import cx_logger
from cortex.lib.concurrency import LockedFile
from cortex.lib.storage import S3, LocalStorage
from cortex.lib.exceptions import CortexException, WithBreak

from cortex.lib.api import (
    PythonPredictorType,
    TensorFlowPredictorType,
    TensorFlowNeuronPredictorType,
    ONNXPredictorType,
    PredictorType,
)
from cortex.lib.model import (
    validate_models_dir_paths,
    validate_model_paths,
)
from cortex.lib.model import (
    ModelsHolder,
    LockedGlobalModelsGC,
    LockedModel,
    CuratedModelResources,
    ModelVersion,
)
from cortex.lib.model import (
    ModelsTree,
    LockedModelsTree,
)

import os
import threading as td
import multiprocessing as mp
import time
import datetime
import glob
import shutil
import itertools


def find_all_models(
    is_dir_used: bool,
    models_dir: str,
    predictor_type: PredictorType,
    s3_paths: List[str],
    s3_model_names: List[str],
) -> Tuple[
    List[str], Dict[str, List[str]], List[str], List[str], List[datetime.datetime], List[str]
]:
    """
    Get updated information on all models that are currently present on the S3 upstreams.
    Information on the available models, versions, last edit times, the subpaths of each model, and so on.

    Args:
        is_dir_used: Whether predictor:models:dir is used or not.
        models_dir: The value of predictor:models:dir in case it's present. Ignored when not required.
        predictor_type: PythonPredictorType, TensorFlowPredictorType, TensorFlowNeuronPredictorType or ONNXPredictorType.
        s3_paths: The model paths as they are specified in predictor:models:paths:model_path/predictor:model_path when predictor:models:paths/predictor:model_path is used. Ignored when not required.
        s3_model_names: The model names as they are specified in predictor:models:paths:name when predictor:models:paths is used or the default name of the model when predictor:model_path is used. Ignored when not required.

    Returns: The tuple with the following elements
        model_names - a list with the names of the models (i.e. bert, gpt-2, etc) and they are unique
        versions - a dictionary with the keys representing the model names and the values being lists of versions that each model has.
          For non-versioned model paths ModelVersion.NOT_PROVIDED, the list will be empty
        model_paths - a list with the prefix of each model
        sub_paths - a list of filepaths for each file of each model all grouped into a single list
        timestamps - a list of timestamps lists representing the last edit time of each versioned model
        bucket_names - a list of the bucket names of each model
    """

    # validate models stored in S3 that were specified with predictor:models:dir field
    if is_dir_used:
        bucket_name, models_path = S3.deconstruct_s3_path(models_dir)
        s3_client = S3(bucket_name, client_config={})
        sub_paths, timestamps = s3_client.search(models_path)
        model_paths, ooa_ids = validate_models_dir_paths(sub_paths, predictor_type, models_path)
        model_names = [os.path.basename(model_path) for model_path in model_paths]

        model_names = list(set(model_names).difference(local_model_names))
        model_paths = [
            model_path for model_path in models_path if os.path.basename(model_path) in model_names
        ]
        bucket_names = len(model_paths) * [bucket_name]

    # validate models stored in S3 that were specified with predictor:models:paths field
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
                s3_client = S3(bucket_name, client_config={})
                sb, model_path_ts = s3_client.search(model_path)
                sub_paths += sb
                timestamps += model_path_ts
                try:
                    ooa_ids.append(validate_model_paths(sub_paths, predictor_type, model_path))
                    model_paths.append(model_path)
                    model_names.append(s3_model_names[idx])
                    bucket_names.append(bucket_name)
                except CortexException:
                    continue

    # determine the detected versions for each model
    # if the model was not versioned, then leave the version list empty
    versions = {}
    for model_path, model_name, model_ooa_ids in zip(model_paths, model_names, ooa_ids):
        if ModelVersion.PROVIDED not in model_ooa_ids:
            versions[model_name] = []
            continue

        model_sub_paths = [os.path.relpath(sub_path, model_path) for sub_path in sub_paths]
        model_versions = [path for path in model_sub_paths if not path.startswith("../")]
        versions[model_name] = model_versions

    # curate timestamps for each versione model
    aux_timestamps = []
    for model_path, model_name in zip(model_paths, model_names):
        model_ts = []
        if len(versions[model_name]) == 0:
            masks = list(map(lambda x: x.startswith(model_path), sub_paths))
            model_ts = [max(itertools.compress(timestamps, masks))]
            continue

        for version in versions[model_name]:
            masks = list(map(lambda x: x.startswith(os.path.join(model_path, version)), sub_paths))
            model_ts.append(max(itertools.compress(timestamps, masks)))
        aux_timestamps.append(model_ts)

    timestamps = aux_timestamps  # type: List[List[datetime.datetime]]

    # model_names - a list with the names of the models (i.e. bert, gpt-2, etc) and they are unique
    # versions - a dictionary with the keys representing the model names and the values being lists of versions that each model has.
    #   For non-versioned model paths ModelVersion.NOT_PROVIDED, the list will be empty
    # model_paths - a list with the prefix of each model
    # sub_paths - a list of filepaths for each file of each model all grouped into a single list
    # timestamps - a list of timestamps lists representing the last edit time of each versioned model

    return model_names, versions, model_paths, sub_paths, timestamps, bucket_names


class FileBasedModelsTreeUpdater(mp.Process):
    """
    Monitors the S3 path(s)/dir and continuously updates the file-based tree.
    The model paths are validated - the bad paths are ignored.
    When a new model is found, it updates the tree and downloads it - likewise when a model is removed.
    """

    def __init__(
        self,
        interval: int,
        api_spec: dict,
        download_dir: str,
        temp_dir: str = "/tmp/cron",
        lock_dir: str = "/run/cron",
    ):
        """
        Args:
            interval: How often to update the models tree. Measured in seconds.
            api_spec: Identical copy of pkg.type.spec.api.API.
            download_dir: Path to where the models are stored.
            temp_dir: Path to where the models are temporarily stored.
            lock_dir: Path to where the resource locks are stored.
        """

        mp.Process.__init__(self)

        self._interval = interval
        self._api_spec = api_spec
        self._download_dir = download_dir
        self._temp_dir = temp_dir
        self._lock_dir = lock_dir

        self._s3_paths = []
        self._spec_models = CuratedModelResources(self._api_spec["curated_model_resources"])
        self._local_model_names = self._spec_models.get_local_model_names()
        self._s3_model_names = self._spec_models.get_s3_model_names()
        for model_name in self._s3_model_names:
            if not self._spec_models.is_local(model_name):
                self._s3_paths.append(curated_model["model_path"])

        if (
            self._api_spec["predictor"]["model_path"] is None
            and self._api_spec["predictor"]["models"]["dir"] is not None
        ):
            self._is_dir_used = True
            self._models_dir = self._api_spec["predictor"]["models"]["dir"]
        else:
            self._is_dir_used = False
            self._models_dir = None

        if self._api_spec["predictor"]["type"] == "python":
            self._predictor_type = PythonPredictorType
        if self._api_spec["predictor"]["type"] == "tensorflow":
            if self._api_spec["compute"]["inf"] > 0:
                self._predictor_type = TensorFlowNeuronPredictorType
            else:
                self._predictor_type = TensorFlowPredictorType
        if self._api_spec["predictor"]["type"] == "onnx":
            self._predictor_type = ONNXPredictorType

        try:
            os.mkdir(self._lock_dir)
        except FileExistsError:
            pass

        self._event_stopper = mp.Event()
        self._stopped = mp.Event()

    def run(self):
        """
        mp.Process-specific method.
        """

        self.logger = cx_logger()
        self._make_local_models_available()
        while not self._event_stopper.is_set():
            self._update_models_tree()
            time.sleep(self._interval)
        self._stopped.set()

    def stop(self, blocking: bool = False):
        """
        Trigger the process of stopping the process.

        Args:
            blocking: Whether to wait until the process is stopped or not.
        """

        self._event_stopper.set()
        if blocking:
            self.join()

    def join(self):
        """
        Block until the process exits.
        """

        while not self._stopped.is_set():
            time.sleep(0.001)

    def _make_local_models_available(self) -> None:
        if self._is_dir_used:
            return

        timestamp_utc = datetime.datetime.now(datetime.timezone.utc).timestamp()
        for local_model_name in self._local_model_names:
            versions = self._spec_models[local_model_name]["versions"]
            if len(versions) == 0:
                resource = local_model_name + "-" + "1" + ".txt"
                with LockedFile(resource, "w") as f:
                    f.write("available " + str(timestamp_utc))
            for ondisk_version in versions:
                resource = local_model_name + "-" + ondisk_version + ".txt"
                with LockedFile(resource, "w") as f:
                    f.write("available " + str(timestamp_utc))

    def _update_models_tree(self) -> None:
        # get updated/validated paths/versions of the S3 models
        model_names, versions, model_paths, sub_paths, timestamps, bucket_names = find_all_models(
            self._is_dir_used,
            self._models_dir,
            self._predictor_type,
            self._s3_paths,
            self._s3_model_names,
        )

        # update models on the local disk if changes have been detected
        # a model is updated if its directory tree has changed, if it's not present or if it doesn't exist on the upstream
        for idx, model_name, bucket_name in enumerate(zip(model_names, bucket_names)):
            self._refresh_model(
                idx, model_name, versions, timestamps[idx], model_paths, sub_paths, bucket_name
            )

        # remove models that no longer appear in model_names
        for models, versions in find_ondisk_models(self._lock_dir, include_versions=True):
            for model_name in models:
                if model_name not in model_names:
                    for ondisk_version in versions:
                        resource = model_name + "-" + ondisk_version + ".txt"
                        ondisk_model_version_path = os.path.join(
                            self._models_dir, model_name, ondisk_version
                        )
                        with LockedFile(resource, "w+") as f:
                            shutil.rmtree(ondisk_model_version_path)
                            f.write("not available")
                    shutil.rmtree(os.path.join(self._models_dir, model_name))

    def _refresh_model(
        self,
        idx: int,
        model_name: str,
        versions: dict,
        timestamps: List[datetime.datetime],
        model_paths: List[str],
        sub_paths: List[str],
        bucket_name: str,
    ) -> None:
        s3_client = S3(bucket_name, client_config={})

        ondisk_model_path = os.path.join(self._download_dir, model_name)
        for version, model_ts in zip(versions[model_name], timestamps):

            # check if a model update is mandated
            update_model = False
            ondisk_model_version_path = os.path.join(ondisk_model_path, version)
            if os.path.exists(ondisk_model_version_path):
                ondisk_paths = glob.glob(ondisk_model_version_path + "*/**", recursive=True)

                s3_model_version_path = os.path.join(model_paths[idx], version)
                s3_paths = [
                    os.path.relpath(sub_path, s3_model_version_path) for sub_path in sub_paths
                ]
                s3_paths = [path for path in s3_paths if not path.startswith("../")]
                s3_paths = util.remove_non_empty_directory_paths(s3_paths)

                if set(ondisk_paths) != set(s3_paths):
                    update_model = True
            else:
                update_model = True

            if update_model:
                # download to a temp directory
                temp_dest = os.path.join(self._temp_dir, model_name, version)
                s3_src = os.path.join(model_paths[idx], version)
                s3_client.download_dir_contents(s3_src, temp_dest)

                # validate the downloaded model
                model_contents = glob.glob(temp_dest + "*/**", recursive=True)
                try:
                    validate_model_paths(model_contents, self._predictor_type, temp_dest)
                    passed_validation = True
                except CortexException:
                    passed_validation = False
                    shutil.rmtree(temp_dest)

                # move the model to its destination directory
                if passed_validation:
                    resource = model_name + "-" + version + ".txt"
                    with LockedFile(resource, "w+") as f:
                        ondisk_model_version = os.path.join(model_paths[idx], version)
                        if os.path.exists(ondisk_model_version):
                            shutil.rmtree(ondisk_model_version)
                        shutil.move(temp_dest, ondisk_model_version)
                        f.write("available " + str(model_ts.timestamp()))

        # remove model versions if they are not found on the upstream
        # except when the model version found on disk is 1 and the number of detected versions on the upstream is 0,
        # thus indicating the 1-version on-disk model must be a model that came without a version
        if os.path.exists(ondisk_model_path):
            ondisk_model_versions = glob.glob(ondisk_model_path + "*/**")
            for ondisk_version in ondisk_model_versions:
                if ondisk_version not in versions[model_name] and (
                    ondisk_version != "1" or len(versions[model_name]) > 0
                ):
                    resource = model_name + "-" + ondisk_version + ".txt"
                    ondisk_model_version_path = os.path.join(ondisk_model_path, ondisk_version)
                    with LockedFile(resource, "w+") as f:
                        shutil.rmtree(ondisk_model_version_path)
                        f.write("not available")

            if len(glob.glob(ondisk_model_path + "*/**")) == 0:
                shutil.rmtree(ondisk_model_path)

        # if it's a non-versioned model ModelVersion.NOT_PROVIDED
        if len(versions[model_name]) == 0:
            # download to a temp directory
            temp_dest = os.path.join(self._temp_dir, model_name)
            s3_client.download_dir_contents(model_paths[idx], temp_dest)

            # validate the downloaded model
            model_contents = glob.glob(temp_dest + "*/**", recursive=True)
            try:
                validate_model_paths(model_contents, self._predictor_type, temp_dest)
                passed_validation = True
            except CortexException:
                passed_validation = False
                shutil.rmtree(temp_dest)

            # move the model to its destination directory
            if passed_validation:
                resource = model_name + "-" + "1" + ".txt"
                with LockedFile(resource, "w+") as f:
                    ondisk_model_version = os.path.join(model_paths[idx], "1")
                    if os.path.exists(ondisk_model_version):
                        shutil.rmtree(ondisk_model_version)
                    shutil.move(temp_dest, ondisk_model_version)
                    f.write("available " + str(timestamps[0].timestamp()))

                # cleanup
                shutil.rmtree(temp_dest)


def find_ondisk_models(
    lock_dir: str, include_versions: bool = False
) -> Union[List[str], Tuple[List[str], List[str]]]:
    """
    Returns all available models from the disk.
    To be used in conjunction with SimpleModelMonitor.

    This function should never be used for determining whether a model has to be loaded or not.
    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        lock_dir: Path to where the resource locks are stored.

    Returns:
        List with the available models from disk. Just the model, no versions.
        Or when include_versions is set, 2 paired lists, one containing the model names and the other one the versions: ["A", "B", "B"] w/ ["1", "1", "2"].
    """

    files = [os.path.basename(file) for file in os.listdir(lock_dir)]
    locks = [f for f in files if f.endswith(".lock")]
    models = []
    versions = []

    for lock in locks:
        _model_name, _model_version = os.path.splitext(lock).split("-")
        status = ""
        with LockedFile(os.path.join(lock_dir, lock), reader_lock=True) as f:
            status = f.read()
        if status == "available":
            if _model_name not in models:
                models.append(_model_name)
                versions.append(_model_version)

    if include_versions:
        return (models, versions)
    else:
        return list(set(models))


def find_ondisk_model_info(lock_dir: str, model_name: str) -> Tuple[List[str], List[int]]:
    """
    Returns all available versions/timestamps of a model from the disk.
    To be used in conjunction with SimpleModelMonitor.

    This function should never be used for determining whether a model has to be loaded or not.
    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        lock_dir: Path to where the resource locks are stored.
        model_name: Name of the model as specified in predictor:models:paths:name, _cortex_default when predictor:model_path is set or the discovered model names when predictor:models:dir is used.

    Returns:
        2-element tuple made of a list with the available versions and a list with the corresponding timestamps for each model. Empty when the model is not available.
    """

    files = [os.path.basename(file) for file in os.listdir(lock_dir)]
    locks = [f for f in files if f.endswith(".lock")]
    versions = []
    timestamps = []

    for lock in locks:
        _model_name, version = os.path.splitext(lock).split("-")
        if _model_name == model_name:
            with LockedFile(os.path.join(lock_dir, lock), reader_lock=True) as f:
                status = f.read()
            if status.startswith("available"):
                current_upstream_ts = int(file_status.split(" ")[1])
                versions.append(version)
                timestamps.append(current_upstream_ts)

    return (versions, timestamps)


class AbstractLoopingThread(td.Thread):
    """
    Abstract class of the td.Thread class.

    Takes a function and keeps calling it in a loop every certain interval.
    """

    def __init__(self, interval: int, runnable: Callable[[], None]):
        td.Thread.__init__(self)

        self._interval = interval
        self._runnable = runnable

        if not callable(self._runnable):
            raise ValueError("runnable parameter must be a callable function")

        self._event_stopper = thread.Event()
        self._stopped = False

    def run(self):
        """
        td.Thread-specific method.
        """

        while not self._event_stopper.is_set():
            self.runnable()
            time.sleep(self._interval)
        self._stopped = True

    def stop(self, blocking: bool = False):
        """
        Stop the thread.

        Args:
            blocking: Whether to wait until the thread is stopped or not.
        """

        self._event_stopper.set()
        if blocking:
            self.join()

    def join(self):
        """
        Block until the thread finishes.
        """

        while not self._stopped:
            time.sleep(0.001)


class ModelsGC(AbstractLoopingThread):
    """
    GC for models loaded into memory and/or stored on disk.

    If the number of models exceeds the cache size, then evict the LRU models.
    Also removes models that are no longer present in the model tree.
    """

    def __init__(
        self, interval: int, api_spec: dict, models: ModelsHolder, tree: ModelsTree,
    ):
        """
        Args:
            interval: How often to update the models tree. Measured in seconds.
            api_spec: Identical copy of pkg.type.spec.api.API.
            models: The object holding all models in memory / on disk.
            tree: Model tree representation of the available models on the S3 upstream.
        """

        td.AbstractLoopingThread.__init__(self, interval, self._run_gc)

        self._api_spec = api_spec
        self._models = models
        self._tree = tree

        self._spec_models = CuratedModelResources(self._api_spec["curated_model_resources"])
        self._local_model_names = self._spec_models.get_local_model_names()
        self._local_model_versions = [
            self._spec_models.get_versions_for(model_name) for model_name in self._local_model_names
        ]
        self._local_model_ids = []
        for model_name, versions in zip(self._local_model_names, self._local_model_versions):
            if len(versions) == 0:
                self._local_model_ids.append(f"{model_name}-1")
                continue
            for version in versions:
                self._local_model_ids.append(f"{model_name}-{version}")

        # run the cron every 10 seconds
        self._lock_timeout = 10.0

        self._event_stopper = thread.Event()
        self._stopped = False

    def _run_gc(self) -> None:
        # are there any models to collect (aka remove) from cache
        with LockedGlobalModelsGC(self._models, "r"):
            collectible = self._models.garbage_collect(
                exclude_disk_model_ids=self._local_model_ids, dry_run=True
            )
        if not collectible:
            return

        # try to grab exclusive access to all models with shared access preference
        # and if it works, remove excess models from cache
        self._models.set_global_preference_policy("r")
        with LockedGlobalModelsGC(self._models, "w", self._lock_timeout) as lg:
            acquired = lg.acquired
            if not acquired:
                raise WithBreak

            self._models.garbage_collect(exclude_disk_model_ids=self._local_model_ids)

        # otherwise, grab exclusive access to all models with exclusive access preference
        # and remove excess models from cache
        self._models.set_global_preference_policy("w")
        with LockedGlobalModelsGC(self._models, "w"):
            self._models.garbage_collect(exclude_disk_model_ids=self._local_model_ids)

        # get available upstream S3 model IDs
        s3_model_names = self._tree.get_model_names()
        s3_model_versions = [self._tree.model_info(model_name) for model_name in s3_model_names]
        s3_model_ids = []
        for model_name, model_versions in zip(s3_model_names, s3_model_versions):
            if len(model_versions) == 0:
                continue
            for model_version in model_versions:
                s3_model_ids.append(f"{model_name}-{model_version}")

        # get model IDs loaded into memory or on disk.
        with LockedGlobalModelsGC(self._models, "r"):
            present_model_ids = self.get_model_ids()

        # remove models that don't exist in the S3 upstream
        ghost_model_ids = list(set(s3_model_ids) - set(present_model_ids))
        for model_id in ghost_model_ids:
            model_name, model_version = model_id.rsplit("-")
            with LockedModel(self._models, "w", model_name, model_version):
                self._models.remove_model(model_name, model_version)


class ModelTreeUpdater(AbstractLoopingThread):
    """
    Model tree updater. Updates a local representation of all available models from the S3 upstreams.
    """

    def __init__(self, interval: int, api_spec: dict, tree: ModelsTree):
        """
        interval: How often to update the models tree. Measured in seconds.
        api_spec: Identical copy of pkg.type.spec.api.API.
        tree: Model tree representation of the available models on the S3 upstream.
        """

        AbstractLoopingThread.__init__(self, interval, self._update_models_tree)

        self._api_spec = api_spec
        self._tree = tree

        self._s3_paths = []
        self._spec_models = CuratedModelResources(self._api_spec["curated_model_resources"])
        self._local_model_names = self._spec_models.get_local_model_names()
        self._s3_model_names = self._spec_models.get_s3_model_names()
        for model_name in self._s3_model_names:
            if not self._spec_models.is_local(model_name):
                self._s3_paths.append(curated_model["model_path"])

        if (
            self._api_spec["predictor"]["model_path"] is None
            and self._api_spec["predictor"]["models"]["dir"] is not None
        ):
            self._is_dir_used = True
            self._models_dir = self._api_spec["predictor"]["models"]["dir"]
        else:
            self._is_dir_used = False
            self._models_dir = None

        if self._api_spec["predictor"]["type"] == "python":
            self._predictor_type = PythonPredictorType
        if self._api_spec["predictor"]["type"] == "tensorflow":
            if self._api_spec["compute"]["inf"] > 0:
                self._predictor_type = TensorFlowNeuronPredictorType
            else:
                self._predictor_type = TensorFlowPredictorType
        if self._api_spec["predictor"]["type"] == "onnx":
            self._predictor_type = ONNXPredictorType

    def _update_models_tree(self) -> None:
        # get updated/validated paths/versions of the S3 models
        model_names, versions, model_paths, sub_paths, timestamps, bucket_names = find_all_models(
            self._is_dir_used,
            self._models_dir,
            self._predictor_type,
            self._s3_paths,
            self._s3_model_names,
        )

        # update model tree
        self._tree.update_models(
            models_names, versions, model_paths, sub_paths, timestamps, bucket_names
        )
