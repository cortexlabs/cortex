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
import time
import datetime
import glob
import shutil
import itertools
import json
import grpc
from concurrent.futures import ThreadPoolExecutor
from typing import Dict, List, Tuple, Any, Union, Callable, Optional

from cortex.lib import util
from cortex.lib.log import cx_logger
from cortex.lib.concurrency import LockedFile, get_locked_files
from cortex.lib.storage import S3, LocalStorage
from cortex.lib.exceptions import CortexException, WithBreak
from cortex.lib.type import (
    PythonPredictorType,
    TensorFlowPredictorType,
    TensorFlowNeuronPredictorType,
    ONNXPredictorType,
    PredictorType,
)

logger = cx_logger()

from cortex.lib.model import (
    TensorFlowServingAPI,
    validate_models_dir_paths,
    validate_model_paths,
    ModelsHolder,
    LockedGlobalModelsGC,
    LockedModel,
    CuratedModelResources,
    ModelVersion,
    ModelsTree,
    LockedModelsTree,
)


def find_all_s3_models(
    is_dir_used: bool,
    models_dir: str,
    predictor_type: PredictorType,
    s3_paths: List[str],
    s3_model_names: List[str],
) -> Tuple[
    List[str],
    Dict[str, List[str]],
    List[str],
    List[List[str]],
    List[List[datetime.datetime]],
    List[str],
]:
    """
    Get updated information on all models that are currently present on the S3 upstreams.
    Information on the available models, versions, last edit times, the subpaths of each model, and so on.

    Args:
        is_dir_used: Whether predictor:models:dir is used or not.
        models_dir: The value of predictor:models:dir in case it's present. Ignored when not required.
        predictor_type: PythonPredictorType, TensorFlowPredictorType, TensorFlowNeuronPredictorType or ONNXPredictorType.
        s3_paths: The S3 model paths as they are specified in predictor:models:paths:model_path/predictor:model_path when predictor:models:paths/predictor:model_path is used. Ignored when not required.
        s3_model_names: The S3 model names as they are specified in predictor:models:paths:name when predictor:models:paths is used or the default name of the model when predictor:model_path is used. Ignored when not required.

    Returns: The tuple with the following elements:
        model_names - a list with the names of the models (i.e. bert, gpt-2, etc) and they are unique
        versions - a dictionary with the keys representing the model names and the values being lists of versions that each model has.
          For non-versioned model paths ModelVersion.NOT_PROVIDED, the list will be empty.
        model_paths - a list with the prefix of each model.
        sub_paths - a list of filepaths lists for each file of each model.
        timestamps - a list of timestamps lists representing the last edit time of each versioned model.
        bucket_names - a list of the bucket names of each model.
    """

    # validate models stored in S3 that were specified with predictor:models:dir field
    if is_dir_used:
        bucket_name, models_path = S3.deconstruct_s3_path(models_dir)
        s3_client = S3(bucket_name, client_config={})
        sub_paths, timestamps = s3_client.search(models_path)
        model_paths, ooa_ids = validate_models_dir_paths(sub_paths, predictor_type, models_path)
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
                try:
                    ooa_ids.append(validate_model_paths(sb, predictor_type, model_path))
                except CortexException:
                    continue
                model_paths.append(model_path)
                model_names.append(s3_model_names[idx])
                bucket_names.append(bucket_name)
                sub_paths += [sb]
                timestamps += [model_path_ts]

    # determine the detected versions for each model
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

    # curate timestamps for each versioned model
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
                self._s3_paths.append(self._spec_models[model_name]["model_path"])

        if (
            self._api_spec["predictor"]["model_path"] is None
            and self._api_spec["predictor"]["models"] is not None
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
        timestamp_utc = datetime.datetime.now(datetime.timezone.utc).timestamp()
        for local_model_name in self._local_model_names:
            versions = self._spec_models[local_model_name]["versions"]
            if len(versions) == 0:
                resource = os.path.join(self._lock_dir, local_model_name + "-" + "1" + ".txt")
                with LockedFile(resource, "w") as f:
                    f.write("available " + str(int(timestamp_utc)))
            for ondisk_version in versions:
                resource = os.path.join(
                    self._lock_dir, local_model_name + "-" + ondisk_version + ".txt"
                )
                with LockedFile(resource, "w") as f:
                    f.write("available " + str(int(timestamp_utc)))

    def _update_models_tree(self) -> None:
        # don't update when the models:dir is a local path
        if self._is_dir_used and not self._models_dir.startswith("s3://"):
            return

        # get updated/validated paths/versions of the S3 models
        (
            model_names,
            versions,
            model_paths,
            sub_paths,
            timestamps,
            bucket_names,
        ) = find_all_s3_models(
            self._is_dir_used,
            self._models_dir,
            self._predictor_type,
            self._s3_paths,
            self._s3_model_names,
        )

        # update models on the local disk if changes have been detected
        # a model is updated if its directory tree has changed, if it's not present or if it doesn't exist on the upstream
        with ThreadPoolExecutor(max_workers=5) as executor:
            futures = []
            for idx, (model_name, bucket_name, bucket_subpaths) in enumerate(
                zip(model_names, bucket_names, sub_paths)
            ):
                futures += [
                    executor.submit(
                        self._refresh_model,
                        idx,
                        model_name,
                        model_paths[idx],
                        versions[model_name],
                        timestamps[idx],
                        bucket_subpaths,
                        bucket_name,
                    )
                ]

            [future.result() for future in futures]

        # remove models that no longer appear in model_names
        for model_name, versions in find_ondisk_models_with_lock(self._lock_dir).items():
            if model_name in model_names or model_name in self._local_model_names:
                continue
            for ondisk_version in versions:
                resource = os.path.join(self._lock_dir, model_name + "-" + ondisk_version + ".txt")
                ondisk_model_version_path = os.path.join(
                    self._download_dir, model_name, ondisk_version
                )
                with LockedFile(resource, "w+") as f:
                    shutil.rmtree(ondisk_model_version_path)
                    f.write("not available")
            shutil.rmtree(os.path.join(self._download_dir, model_name))

    def _refresh_model(
        self,
        idx: int,
        model_name: str,
        model_path: str,
        versions: List[str],
        timestamps: List[datetime.datetime],
        sub_paths: List[str],
        bucket_name: str,
    ) -> None:
        s3_client = S3(bucket_name, client_config={})

        ondisk_model_path = os.path.join(self._download_dir, model_name)
        for version, model_ts in zip(versions, timestamps):

            # for the lock file
            resource = os.path.join(self._lock_dir, model_name + "-" + version + ".txt")

            # check if a model update is mandated
            update_model = False
            ondisk_model_version_path = os.path.join(ondisk_model_path, version)
            if os.path.exists(ondisk_model_version_path):
                local_paths = glob.glob(ondisk_model_version_path + "*/**", recursive=True)
                local_paths = [
                    os.path.relpath(local_path, ondisk_model_version_path)
                    for local_path in local_paths
                ]
                local_paths = [path for path in local_paths if not path.startswith("../")]
                local_paths = util.remove_non_empty_directory_paths(local_paths)

                s3_model_version_path = os.path.join(model_path, version)
                s3_paths = [
                    os.path.relpath(sub_path, s3_model_version_path) for sub_path in sub_paths
                ]
                s3_paths = [path for path in s3_paths if not path.startswith("../")]
                s3_paths = util.remove_non_empty_directory_paths(s3_paths)

                # update if the paths don't match
                if set(local_paths) != set(s3_paths):
                    update_model = True

                # update if the timestamp is newer
                with LockedFile(resource, "r", reader_lock=True) as f:
                    file_status = f.read()
                    if file_status == "" or file_status == "not available":
                        raise WithBreak
                    current_model_ts = int(file_status.split(" ")[1])
                    if current_model_ts < int(model_ts.timestamp()):
                        update_model = True
            else:
                update_model = True

            if update_model:
                # download to a temp directory
                temp_dest = os.path.join(self._temp_dir, model_name, version)
                s3_src = os.path.join(model_path, version)
                s3_client.download_dir_contents(s3_src, temp_dest)

                # validate the downloaded model
                model_contents = glob.glob(temp_dest + "*/**", recursive=True)
                model_contents = util.remove_non_empty_directory_paths(model_contents)
                try:
                    validate_model_paths(model_contents, self._predictor_type, temp_dest)
                    passed_validation = True
                except CortexException:
                    passed_validation = False
                    shutil.rmtree(temp_dest)

                # move the model to its destination directory
                if passed_validation:
                    with LockedFile(resource, "w+") as f:
                        if os.path.exists(ondisk_model_version_path):
                            shutil.rmtree(ondisk_model_version_path)
                        shutil.move(temp_dest, ondisk_model_version_path)
                        f.write("available " + str(int(model_ts.timestamp())))

        # remove the temp model directory if it exists
        model_temp_dest = os.path.join(self._temp_dir, model_name)
        if os.path.exists(model_temp_dest):
            os.rmdir(model_temp_dest)

        # remove model versions if they are not found on the upstream
        # except when the model version found on disk is 1 and the number of detected versions on the upstream is 0,
        # thus indicating the 1-version on-disk model must be a model that came without a version
        if os.path.exists(ondisk_model_path):
            ondisk_model_versions = glob.glob(ondisk_model_path + "*/**")
            ondisk_model_versions = [
                os.path.relpath(path, ondisk_model_path) for path in ondisk_model_versions
            ]
            for ondisk_version in ondisk_model_versions:
                if ondisk_version not in versions and (ondisk_version != "1" or len(versions) > 0):
                    resource = os.path.join(
                        self._lock_dir, model_name + "-" + ondisk_version + ".txt"
                    )
                    ondisk_model_version_path = os.path.join(ondisk_model_path, ondisk_version)
                    with LockedFile(resource, "w+") as f:
                        shutil.rmtree(ondisk_model_version_path)
                        f.write("not available")

            # remove the model directory if there are no models left
            if len(glob.glob(ondisk_model_path + "*/**")) == 0:
                shutil.rmtree(ondisk_model_path)

        # if it's a non-versioned model ModelVersion.NOT_PROVIDED
        if len(versions) == 0 and len(sub_paths) > 0:

            # for the lock file
            resource = os.path.join(self._lock_dir, model_name + "-" + "1" + ".txt")
            model_ts = int(timestamps[0].timestamp())

            # check if a model update is mandated
            update_model = False
            ondisk_model_version_path = os.path.join(ondisk_model_path, "1")
            if os.path.exists(ondisk_model_version_path):
                local_paths = glob.glob(ondisk_model_version_path + "*/**", recursive=True)
                local_paths = [
                    os.path.relpath(local_path, ondisk_model_version_path)
                    for local_path in local_paths
                ]
                local_paths = [path for path in local_paths if not path.startswith("../")]
                local_paths = util.remove_non_empty_directory_paths(local_paths)

                s3_model_version_path = model_path
                s3_paths = [
                    os.path.relpath(sub_path, s3_model_version_path) for sub_path in sub_paths
                ]
                s3_paths = [path for path in s3_paths if not path.startswith("../")]
                s3_paths = util.remove_non_empty_directory_paths(s3_paths)

                # update if the paths don't match
                if set(local_paths) != set(s3_paths):
                    update_model = True

                # update if the timestamp is newer
                with LockedFile(resource, "r", reader_lock=True) as f:
                    file_status = f.read()
                    if file_status == "" or file_status == "not available":
                        raise WithBreak()
                    current_model_ts = int(file_status.split(" ")[1])
                    if current_model_ts < model_ts:
                        update_model = True
            else:
                update_model = True

            if not update_model:
                return

            # download to a temp directory
            temp_dest = os.path.join(self._temp_dir, model_name)
            s3_client.download_dir_contents(model_path, temp_dest)

            # validate the downloaded model
            model_contents = glob.glob(temp_dest + "*/**", recursive=True)
            model_contents = util.remove_non_empty_directory_paths(model_contents)
            try:
                validate_model_paths(model_contents, self._predictor_type, temp_dest)
                passed_validation = True
            except CortexException:
                passed_validation = False
                shutil.rmtree(temp_dest)

            # move the model to its destination directory
            if passed_validation:
                with LockedFile(resource, "w+") as f:
                    if os.path.exists(ondisk_model_version_path):
                        shutil.rmtree(ondisk_model_version_path)
                    shutil.move(temp_dest, ondisk_model_version_path)
                    f.write("available " + str(model_ts))


def find_ondisk_models_with_lock(lock_dir: str) -> Dict[str, List[str]]:
    """
    Returns all available models from the disk.
    To be used in conjunction with FileBasedModelsTreeUpdater.

    This function should never be used for determining whether a model has to be loaded or not.
    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        lock_dir: Path to where the resource locks are stored.

    Returns:
        Dictionary with available model names and their associated versions.
        {
            "model-A": [177, 245, 247],
            "model-B": [1],
            ...
        }
    """
    models = {}

    for locked_file in get_locked_files(lock_dir):
        _model_name, _model_version = os.path.splitext(locked_file)[0].rsplit("-", maxsplit=1)
        with LockedFile(os.path.join(lock_dir, locked_file), "r", reader_lock=True) as f:
            status = f.read()
        if status.startswith("available"):
            if _model_name not in models:
                models[_model_name] = [_model_version]
            else:
                models[_model_name] += [_model_version]

    return models


def find_ondisk_model_info(lock_dir: str, model_name: str) -> Tuple[List[str], List[int]]:
    """
    Returns all available versions/timestamps of a model from the disk.

    This function should never be used for determining whether a model has to be loaded or not.
    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        lock_dir: Path to where the resource locks are stored.
        model_name: Name of the model as specified in predictor:models:paths:name, _cortex_default when predictor:model_path is set or the discovered model names when predictor:models:dir is used.

    Returns:
        2-element tuple made of a list with the available versions and a list with the corresponding timestamps for each model. Empty when the model is not available.
    """
    versions = []
    timestamps = []

    for locked_file in get_locked_files(lock_dir):
        _model_name, _model_version = os.path.splitext(locked_file)[0].rsplit("-", maxsplit=1)
        if _model_name != model_name:
            continue

        with LockedFile(os.path.join(lock_dir, locked_file), "r", reader_lock=True) as f:
            status = f.read()
        if not status.startswith("available"):
            continue

        current_upstream_ts = int(status.split(" ")[1])
        timestamps.append(current_upstream_ts)
        versions.append(_model_version)

    return (versions, timestamps)


class TFSModelLoader(mp.Process):
    """
    Monitors the S3 path(s)/dir and continuously updates the models on TFS.
    The model paths are validated - the bad paths are ignored.
    When a new model is found, it updates the tree, downloads it and loads it into memory - likewise when a model is removed.
    """

    def __init__(
        self,
        interval: int,
        api_spec: dict,
        address: str,
        tfs_model_dir: str,
        download_dir: str,
        temp_dir: str = "/tmp/cron",
        lock_dir: str = "/run/cron",
    ):
        """
        Args:
            interval: How often to update the models tree. Measured in seconds.
            api_spec: Identical copy of pkg.type.spec.api.API.
            address: An address with the "host:port" format to where TFS is located.
            download_dir: Path to where the models are stored.
            tfs_model_dir: Path to where the models are stored within the TFS container.
            lock_dir: Directory in which model timestamps are stored.
        """

        mp.Process.__init__(self)

        self._interval = interval
        self._api_spec = api_spec
        self._tfs_model_dir = tfs_model_dir
        self._download_dir = download_dir
        self._temp_dir = temp_dir
        self._lock_dir = lock_dir

        self._tfs_address = address
        self._client = TensorFlowServingAPI(address)
        self._old_ts_state = {}

        self._s3_paths = []
        self._spec_models = CuratedModelResources(self._api_spec["curated_model_resources"])
        self._local_model_names = self._spec_models.get_local_model_names()
        self._s3_model_names = self._spec_models.get_s3_model_names()
        for model_name in self._s3_model_names:
            if not self._spec_models.is_local(model_name):
                self._s3_paths.append(self._spec_models[model_name]["model_path"])

        if (
            self._api_spec["predictor"]["model_path"] is None
            and self._api_spec["predictor"]["models"] is not None
            and self._api_spec["predictor"]["models"]["dir"] is not None
        ):
            self._is_dir_used = True
            self._models_dir = self._api_spec["predictor"]["models"]["dir"]
        else:
            self._is_dir_used = False
            self._models_dir = None

        if self._api_spec["predictor"]["type"] == "tensorflow":
            if self._api_spec["compute"]["inf"] > 0:
                self._predictor_type = TensorFlowNeuronPredictorType
            else:
                self._predictor_type = TensorFlowPredictorType
        else:
            raise CortexException(
                "'tensorflow' predictor type is the only allowed type for this cron"
            )

        self._event_stopper = mp.Event()
        self._stopped = mp.Event()

    def run(self):
        """
        mp.Process-specific method.
        """

        self.logger = cx_logger()
        while not self._event_stopper.is_set():
            self._update_models()
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

    def _update_models(self) -> None:
        # don't update when the models:dir is a local path
        if self._is_dir_used and not self._models_dir.startswith("s3://"):
            return

        # get updated/validated paths/versions of the S3 models
        (
            model_names,
            versions,
            model_paths,
            sub_paths,
            timestamps,
            bucket_names,
        ) = find_all_s3_models(
            self._is_dir_used,
            self._models_dir,
            self._predictor_type,
            self._s3_paths,
            self._s3_model_names,
        )

        # TODO download models concurrently

        # update models on the local disk if changes have been detected
        # a model is updated if its directory tree has changed, if it's not present or if it doesn't exist on the upstream
        for idx, (model_name, bucket_name, bucket_sub_paths) in enumerate(
            zip(model_names, bucket_names, sub_paths)
        ):
            self._refresh_model(
                idx,
                model_name,
                model_paths[idx],
                versions[model_name],
                timestamps[idx],
                bucket_sub_paths,
                bucket_name,
            )

        # remove models that no longer appear in model_names
        for model_name, model_versions in find_ondisk_models(self._download_dir).items():
            if model_name in model_names or model_name in self._local_model_names:
                continue
            for ondisk_version in model_versions:
                ondisk_model_version_path = os.path.join(
                    self._download_dir, model_name, ondisk_version
                )
                shutil.rmtree(ondisk_model_version_path)
            shutil.rmtree(os.path.join(self._download_dir, model_name))
            self._client.remove_models([model_name], [model_versions])

        # TODO move the code within the for loop in a separate method

        # check tfs connection
        tfs_unresponsive = not self._client.is_tfs_accessible()
        if tfs_unresponsive:
            self.cleanup_when_tfs_unresponsive()
            return

        # remove versioned models from TFS that no longer exist on disk
        tfs_model_ids = self._client.get_registered_model_ids()
        ondisk_models = find_ondisk_models(self._download_dir)
        ondisk_model_ids = []
        for model_name, model_versions in ondisk_models.items():
            for model_version in model_versions:
                ondisk_model_ids.append(f"{model_name}-{model_version}")
        for tfs_model_id in tfs_model_ids:
            if tfs_model_id not in ondisk_model_ids:
                try:
                    model_name, model_version = tfs_model_id.rsplit("-", maxsplit=1)
                    self._client.remove_single_model(model_name, model_version)
                    logger.info(
                        "model '{}' of version '{}' has been unloaded".format(
                            model_name, model_version
                        )
                    )
                except gprc.RpcError as error:
                    if error.code() == grpc.StatusCode.UNAVAILABLE:
                        logger.warning(
                            "TFS server unresponsive after trying to load model '{}' of version '{}': ".format(
                                model_name, model_version, str(e)
                            )
                        )
                    self.cleanup_when_tfs_unresponsive()
                    return

        # update TFS models
        current_ts_state = {}
        for model_name, model_versions in ondisk_models.items():

            # get the right model version with respect to the model ts order
            model_timestamps = timestamps[model_names.index(model_name)]
            filtered_model_versions = []
            for idx, model_ts in enumerate(model_timestamps):
                if versions[model_name][idx] in model_versions:
                    filtered_model_versions.append(versions[model_name][idx])

            for model_version, model_ts in zip(filtered_model_versions, model_timestamps):
                model_ts = int(model_ts.timestamp())

                # remove outdated model
                model_id = f"{model_name}-{model_version}"
                model_reloaded = False
                first_time_load = False
                if model_id in self._old_ts_state and self._old_ts_state[model_id] < model_ts:
                    try:
                        self._client.remove_single_model(model_name, model_version)
                    except gprc.RpcError as error:
                        if error.code() == grpc.StatusCode.UNAVAILABLE:
                            logger.warning(
                                "TFS server unresponsive after trying to unload model '{}' of version '{}': ".format(
                                    model_name, model_version, str(e)
                                )
                            )
                        logger.warning("TFS server is unresponsive")
                        return
                    model_reloaded = True
                elif model_id not in self._old_ts_state:
                    first_time_load = True

                if not model_reloaded and not first_time_load:
                    continue

                # load model
                model_disk_path = os.path.join(self._tfs_model_dir, model_name)
                try:
                    self._client.add_single_model(
                        model_name,
                        model_version,
                        model_disk_path,
                        self._determine_model_signature_key(model_name),
                        timeout=30.0,
                    )
                except Exception as e:
                    try:
                        self._client.remove_single_model(model_name, model_version)
                        logger.warning(
                            "model '{}' of version '{}' couldn't be loaded: {}".format(
                                model_name, model_version, str(e)
                            )
                        )
                    except grpc.RpcError as error:
                        if error.code() == grpc.StatusCode.UNAVAILABLE:
                            logger.warning(
                                "TFS server unresponsive after trying to load model '{}' of version '{}': ".format(
                                    model_name, model_version, str(e)
                                )
                            )
                        self.cleanup_when_tfs_unresponsive()
                        return

                    model_reloaded = False
                    first_time_load = False

                # save timestamp of loaded model
                if model_reloaded:
                    current_ts_state[model_id] = model_ts
                    logger.info(
                        "model '{}' of version '{}' has been reloaded".format(
                            model_name, model_version
                        )
                    )
                elif first_time_load:
                    current_ts_state[model_id] = model_ts
                    logger.info(
                        "model '{}' of version '{}' has been loaded".format(
                            model_name, model_version
                        )
                    )

        # save model timestamp states
        for model_id, ts in current_ts_state.items():
            self._old_ts_state[model_id] = ts

        # remove model timestamps that no longer exist
        loaded_model_ids = self._client.models.keys()
        aux_ts_state = self._old_ts_state.copy()
        for model_id in self._old_ts_state.keys():
            if model_id not in loaded_model_ids:
                del aux_ts_state[model_id]
        self._old_ts_state = aux_ts_state

        # save model timestamp states to disk
        # could be cast to a short-lived thread
        # required for printing the model stats when cortex getting
        resource = os.path.join(self._lock_dir, "model_timestamps.json")
        with LockedFile(resource, "w") as f:
            json.dump(self._old_ts_state, f, indent=2)

        # save model stats for TFS to disk
        resource = os.path.join(self._lock_dir, "models_tfs.json")
        with LockedFile(resource, "w") as f:
            json.dump(self._client.models, f, indent=2)

    def _refresh_model(
        self,
        idx: int,
        model_name: str,
        model_path: str,
        versions: List[str],
        timestamps: List[datetime.datetime],
        sub_paths: List[str],
        bucket_name: str,
    ) -> None:
        s3_client = S3(bucket_name, client_config={})

        ondisk_model_path = os.path.join(self._download_dir, model_name)
        for version, model_ts in zip(versions, timestamps):

            # check if a model update is mandated
            update_model = False
            ondisk_model_version_path = os.path.join(ondisk_model_path, version)
            if os.path.exists(ondisk_model_version_path):
                local_paths = glob.glob(ondisk_model_version_path + "*/**", recursive=True)
                local_paths = [
                    os.path.relpath(local_path, ondisk_model_version_path)
                    for local_path in local_paths
                ]
                local_paths = [path for path in local_paths if not path.startswith("../")]
                local_paths = util.remove_non_empty_directory_paths(local_paths)

                s3_model_version_path = os.path.join(model_path, version)
                s3_paths = [
                    os.path.relpath(sub_path, s3_model_version_path) for sub_path in sub_paths
                ]
                s3_paths = [path for path in s3_paths if not path.startswith("../")]
                s3_paths = util.remove_non_empty_directory_paths(s3_paths)

                if set(local_paths) != set(s3_paths):
                    update_model = True

                model_id = f"{model_name}-{version}"
                if self._is_this_a_newer_model_id(model_id, int(model_ts.timestamp())):
                    update_model = True
            else:
                update_model = True

            if update_model:
                # download to a temp directory
                temp_dest = os.path.join(self._temp_dir, model_name, version)
                s3_src = os.path.join(model_path, version)
                s3_client.download_dir_contents(s3_src, temp_dest)

                # validate the downloaded model
                model_contents = glob.glob(temp_dest + "*/**", recursive=True)
                model_contents = util.remove_non_empty_directory_paths(model_contents)
                try:
                    validate_model_paths(model_contents, self._predictor_type, temp_dest)
                    passed_validation = True
                except CortexException:
                    passed_validation = False
                    shutil.rmtree(temp_dest)

                # move the model to its destination directory
                if passed_validation:
                    if os.path.exists(ondisk_model_version_path):
                        shutil.rmtree(ondisk_model_version_path)
                    shutil.move(temp_dest, ondisk_model_version_path)

        # remove the temp model directory if it exists
        model_temp_dest = os.path.join(self._temp_dir, model_name)
        if os.path.exists(model_temp_dest):
            os.rmdir(model_temp_dest)

        # remove model versions if they are not found on the upstream
        # except when the model version found on disk is 1 and the number of detected versions on the upstream is 0,
        # thus indicating the 1-version on-disk model must be a model that came without a version
        if os.path.exists(ondisk_model_path):
            ondisk_model_versions = glob.glob(ondisk_model_path + "*/**")
            ondisk_model_versions = [
                os.path.relpath(path, ondisk_model_path) for path in ondisk_model_versions
            ]
            for ondisk_version in ondisk_model_versions:
                if ondisk_version not in versions and (ondisk_version != "1" or len(versions) > 0):
                    ondisk_model_version_path = os.path.join(ondisk_model_path, ondisk_version)
                    shutil.rmtree(ondisk_model_version_path)

            if len(glob.glob(ondisk_model_path + "*/**")) == 0:
                shutil.rmtree(ondisk_model_path)

        # if it's a non-versioned model ModelVersion.NOT_PROVIDED
        if len(versions) == 0 and len(sub_paths) > 0:

            model_ts = timestamps[0]

            # check if a model update is mandated
            update_model = False
            ondisk_model_version_path = os.path.join(ondisk_model_path, "1")
            if os.path.exists(ondisk_model_version_path):
                local_paths = glob.glob(ondisk_model_version_path + "*/**", recursive=True)
                local_paths = [
                    os.path.relpath(local_path, ondisk_model_version_path)
                    for local_path in local_paths
                ]
                local_paths = [path for path in local_paths if not path.startswith("../")]
                local_paths = util.remove_non_empty_directory_paths(local_paths)

                s3_model_version_path = model_path
                s3_paths = [
                    os.path.relpath(sub_path, s3_model_version_path) for sub_path in sub_paths
                ]
                s3_paths = [path for path in s3_paths if not path.startswith("../")]
                s3_paths = util.remove_non_empty_directory_paths(s3_paths)

                # update if the paths don't match
                if set(local_paths) != set(s3_paths):
                    update_model = True

                model_id = f"{model_name}-1"
                if self._is_this_a_newer_model_id(model_id, int(model_ts.timestamp())):
                    update_model = True
            else:
                update_model = True

            if not update_model:
                return

            # download to a temp directory
            temp_dest = os.path.join(self._temp_dir, model_name)
            s3_client.download_dir_contents(model_path, temp_dest)

            # validate the downloaded model
            model_contents = glob.glob(temp_dest + "*/**", recursive=True)
            model_contents = util.remove_non_empty_directory_paths(model_contents)
            try:
                validate_model_paths(model_contents, self._predictor_type, temp_dest)
                passed_validation = True
            except CortexException:
                passed_validation = False
                shutil.rmtree(temp_dest)

            # move the model to its destination directory
            if passed_validation:
                if os.path.exists(ondisk_model_version_path):
                    shutil.rmtree(ondisk_model_version_path)
                shutil.move(temp_dest, ondisk_model_version_path)

    def _is_this_a_newer_model_id(self, model_id: str, timestamp: int) -> bool:
        return model_id in self._old_ts_state and self._old_ts_state[model_id] < timestamp

    def _determine_model_signature_key(self, model_name: str) -> Optional[str]:
        if self._models_dir:
            signature_key = self._api_spec["predictor"]["models"]["signature_key"]
        else:
            signature_key = self._spec_model_names[model_name]["signature_key"]

        return signature_key

    def cleanup_when_tfs_unresponsive(self):
        logger.warning("TFS server is unresponsive")

        self._client = TensorFlowServingAPI(self._tfs_address)

        resource = os.path.join(self._lock_dir, "models_tfs.json")
        with LockedFile(resource, "w") as f:
            json.dump(self._client.models, f, indent=2)


def find_ondisk_models(models_dir: str) -> Dict[str, List[str]]:
    """
    Returns all available models from the disk.
    To be used in conjunction with TFSModelLoader.

    This function should never be used for determining whether a model has to be loaded or not.
    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        models_dir: Path to where the models are stored.

    Returns:
        Dictionary with available model names and their associated versions.
        {
            "model-A": [177, 245, 247],
            "model-B": [1],
            ...
        }
    """

    models = {}
    model_names = [os.path.basename(file) for file in os.listdir(models_dir)]

    for model_name in model_names:
        model_versions = os.listdir(os.path.join(models_dir, model_name))
        models[model_name] = model_versions

    return models


class AbstractLoopingThread(td.Thread):
    """
    Abstract class of the td.Thread class.

    Takes a function and keeps calling it in a loop every certain interval.
    """

    def __init__(self, interval: int, runnable: Callable[[], None]):
        td.Thread.__init__(self, daemon=True)

        self._interval = interval
        self._runnable = runnable

        if not callable(self._runnable):
            raise ValueError("runnable parameter must be a callable function")

        self._event_stopper = td.Event()
        self._stopped = False

    def run(self):
        """
        td.Thread-specific method.
        """

        while not self._event_stopper.is_set():
            self._runnable()
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
        self,
        interval: int,
        api_spec: dict,
        models: ModelsHolder,
        tree: ModelsTree,
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
            self._remove_stale_models()
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
        if acquired:
            self._models.set_global_preference_policy("w")
            with LockedGlobalModelsGC(self._models, "w"):
                self._models.garbage_collect(exclude_disk_model_ids=self._local_model_ids)

        self._remove_stale_models()

    def _remove_stale_models(self) -> None:

        # get available upstream S3 model IDs
        s3_model_names = self._tree.get_model_names()
        s3_model_versions = [
            self._tree.model_info(model_name)["versions"] for model_name in s3_model_names
        ]
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
            model_name, model_version = model_id.rsplit("-", maxsplit=1)
            with LockedModel(self._models, "w", model_name, model_version):
                self._models.remove_model(model_name, model_version)


class ModelTreeUpdater(AbstractLoopingThread):
    """
    Model tree updater. Updates a local representation of all available models from the S3 upstreams.
    """

    def __init__(self, interval: int, api_spec: dict, tree: ModelsTree, ondisk_models_dir: str):
        """
        Args:
            interval: How often to update the models tree. Measured in seconds.
            api_spec: Identical copy of pkg.type.spec.api.API.
            tree: Model tree representation of the available models on the S3 upstream.
            ondisk_models_dir: Where the models are stored on disk. Necessary when local models are used.
        """

        AbstractLoopingThread.__init__(self, interval, self._update_models_tree)

        self._api_spec = api_spec
        self._tree = tree
        self._ondisk_models_dir = ondisk_models_dir

        self._s3_paths = []
        self._spec_models = CuratedModelResources(self._api_spec["curated_model_resources"])
        self._s3_model_names = self._spec_models.get_s3_model_names()
        for model_name in self._s3_model_names:
            if not self._spec_models.is_local(model_name):
                self._s3_paths.append(self._spec_models[model_name]["model_path"])

        if (
            self._api_spec["predictor"]["model_path"] is None
            and self._api_spec["predictor"]["models"] is not None
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

        self._make_local_models_available()

    def _make_local_models_available(self):
        timestamp_utc = datetime.datetime.now(datetime.timezone.utc)

        for model_name in self._spec_models.get_local_model_names():
            model = self._spec_models[model_name]

            if len(model["versions"]) == 0:
                model_version = "1"
                ondisk_model_version_path = os.path.join(
                    self._ondisk_models_dir, model_name, model_version
                )
                ondisk_paths = glob.glob(ondisk_model_version_path + "*/**", recursive=True)
                ondisk_paths = util.remove_non_empty_directory_paths(ondisk_paths)
                self._tree.update_model(
                    bucket="",
                    model_name=model_name,
                    model_version=model_version,
                    model_path=ondisk_model_version_path,
                    sub_paths=ondisk_paths,
                    timestamp=timestamp_utc,
                    tree_removable=False,
                )

            for model_version in model["versions"]:
                ondisk_model_version_path = os.path.join(
                    self._ondisk_models_dir, model_name, model_version
                )
                ondisk_paths = glob.glob(ondisk_model_version_path + "*/**", recursive=True)
                ondisk_paths = util.remove_non_empty_directory_paths(ondisk_paths)
                self._tree.update_model(
                    bucket="",
                    model_name=model_name,
                    model_version=model_version,
                    model_path=ondisk_model_version_path,
                    sub_paths=ondisk_paths,
                    timestamp=timestamp_utc,
                    tree_removable=False,
                )

    def _update_models_tree(self) -> None:
        # don't update when the models:dir is a local path
        if self._is_dir_used and not self._models_dir.startswith("s3://"):
            return

        # get updated/validated paths/versions of the S3 models
        (
            model_names,
            versions,
            model_paths,
            sub_paths,
            timestamps,
            bucket_names,
        ) = find_all_s3_models(
            self._is_dir_used,
            self._models_dir,
            self._predictor_type,
            self._s3_paths,
            self._s3_model_names,
        )

        # update model tree
        self._tree.update_models(
            model_names, versions, model_paths, sub_paths, timestamps, bucket_names
        )


class ModelPreloader(AbstractLoopingThread):
    """
    Model preloader for models that had only been called using either tag ("latest" or "highest") for at least a certain number of times.
    """

    def __init__(self, interval: int, caching: bool, models: ModelsHolder, tree: ModelsTree):
        """
        Args:
            interval: How often to update the models tree. Measured in seconds.
            caching: Whether caching is enabled or not. Can only set caching to True.
            models: The object holding all models in memory / on disk.
            tree: Model tree representation of the available models on the S3 upstream.
        """
        AbstractLoopingThread.__init__(self, interval, self._preload_models)

        if not caching:
            raise NotImplementedError(
                "the model preloader hasn't been implement for when the caching is disabled"
            )

        if tree is not None and caching is False:
            raise ValueError("tree must be None when caching is disabled")

        self._cache_enabled = caching
        self._models = models

        if caching:
            self._tree = tree

    def _preload_models(self) -> None:
        # preload new models that have only been called using the "latest" tag
        latest_model_names, ts = self._models.get_model_names_by_tag_count("latest", 10)
        for model_name, model_ts in zip(latest_model_names, ts):
            if self._cache_enabled:
                self._preload_model_when_caching("timestamps", model_name, model_ts)
            else:
                self._preload_model_when_not_caching("timestamps", model_name, model_ts)

        # preload new models that have only been called using the "highest" tag
        highest_model_names, ts = self._models.get_model_names_by_tag_count("highest", 10)
        for model_name, model_ts in zip(latest_model_names, ts):
            if self._cache_enabled:
                self._preload_model_when_caching("versions", model_name, model_ts)
            else:
                self._preload_model_when_not_caching("timestamps", model_name, model_ts)

    def _preload_model_when_caching(self, dict_key: str, model_name: str, model_ts: int) -> None:

        # get the latest model info according to the model name and dict key selector
        model_info = self._tree.model_info(model_name)
        idx = model_info[dict_key].index(max(model_info[dict_key]))
        model_version = model_info["versions"][idx]
        model_path = model_info["model_paths"][idx]

        # verify if the latest model has to be preloaded or not (maybe it already got loaded)
        update_model = False
        with LockedModel(self._models, "r", model_name, model_version):
            status, current_ts = self._models.has_model(model_name, model_version)
            if current_ts >= model_ts:
                raise WithBreak
            update_model = True

        if not update_model:
            return

        # preload the model
        with LockedModel(self._models, "w", model_name, model_version):
            status, current_ts = self._models.has_model(model_name, model_version)
            if current_ts >= model_ts:
                raise WithBreak

            if status == "in-memory":
                self._models.remove_model(model_name, model_version)
                self._models.download_model(
                    model_info["bucket"], model_name, model_version, model_path
                )

            self._models.load_model(model_name, model_version, model_ts, ["latest", "highest"])

    def _preload_model_when_not_caching(
        self, dict_key: str, model_name: str, model_ts: int
    ) -> None:

        # TODO implement "latest"/"highest" model preloader when caching is disabled
        pass
