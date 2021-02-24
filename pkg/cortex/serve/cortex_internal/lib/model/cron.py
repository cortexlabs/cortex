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

import os
import threading as td
import multiprocessing as mp
import time
import datetime
import glob
import shutil
import json
import grpc
import copy
from concurrent.futures import ThreadPoolExecutor
from typing import Dict, List, Tuple, Any, Union, Callable, Optional

from cortex_internal.lib import util
from cortex_internal.lib.concurrency import LockedFile, get_locked_files
from cortex_internal.lib.storage import S3, GCS
from cortex_internal.lib.exceptions import CortexException, WithBreak
from cortex_internal.lib.type import (
    predictor_type_from_api_spec,
    PythonPredictorType,
    TensorFlowPredictorType,
    TensorFlowNeuronPredictorType,
)

from cortex_internal.lib.model import (
    find_all_cloud_models,
    validate_model_paths,
    TensorFlowServingAPI,
    TensorFlowServingAPIClones,
    ModelsHolder,
    ids_to_models,
    LockedGlobalModelsGC,
    LockedModel,
    get_models_from_api_spec,
    ModelsTree,
)
from cortex_internal.lib.telemetry import get_default_tags, init_sentry
from cortex_internal.lib.log import configure_logger

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


class AbstractLoopingThread(td.Thread):
    """
    Abstract class of the td.Thread class.

    Takes a method and keeps calling it in a loop every certain interval.
    """

    def __init__(self, interval: int, runnable: Callable[[], None]):
        td.Thread.__init__(self, daemon=True)

        self._interval = interval
        self._runnable = runnable

        if not callable(self._runnable):
            raise ValueError("runnable parameter must be a callable method")

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

        mp.Process.__init__(self, daemon=True)

        self._interval = interval
        self._api_spec = api_spec
        self._download_dir = download_dir
        self._temp_dir = temp_dir
        self._lock_dir = lock_dir

        self._s3_paths = []
        self._spec_models = get_models_from_api_spec(self._api_spec)
        self._local_model_names = self._spec_models.get_local_model_names()
        self._cloud_model_names = self._spec_models.get_cloud_model_names()
        for model_name in self._cloud_model_names:
            self._s3_paths.append(self._spec_models[model_name]["path"])

        self._predictor_type = predictor_type_from_api_spec(self._api_spec)

        if (
            self._predictor_type == PythonPredictorType
            and self._api_spec["predictor"]["multi_model_reloading"]
        ):
            models = self._api_spec["predictor"]["multi_model_reloading"]
        elif self._predictor_type != PythonPredictorType:
            models = self._api_spec["predictor"]["models"]
        else:
            models = None

        if models is None:
            raise CortexException("no specified model")

        if models["dir"] is not None:
            self._is_dir_used = True
            self._models_dir = models["dir"]
        else:
            self._is_dir_used = False
            self._models_dir = None

        try:
            os.mkdir(self._lock_dir)
        except FileExistsError:
            pass

        self._ran_once = mp.Event()
        self._event_stopper = mp.Event()
        self._stopped = mp.Event()

    def run(self):
        """
        mp.Process-specific method.
        """

        init_sentry(tags=get_default_tags())
        self._make_local_models_available()
        while not self._event_stopper.is_set():
            self._update_models_tree()
            if not self._ran_once.is_set():
                self._ran_once.set()
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

    def ran_once(self) -> bool:
        """
        Tells whether the FileBasedModelsTreeUpdater loop has run at least once.
        """

        return self._ran_once.is_set()

    def _make_local_models_available(self) -> None:
        """
        Make local models (provided through models:path, models:paths or models:dir fields) available on disk.
        """

        timestamp_utc = datetime.datetime.now(datetime.timezone.utc).timestamp()

        if len(self._local_model_names) == 1:
            message = "local model "
        elif len(self._local_model_names) > 1:
            message = "local models "
        else:
            return

        for idx, local_model_name in enumerate(self._local_model_names):
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

            message += f"{local_model_name} "
            if len(versions) == 1:
                message += f"(version {versions[0]})"
            elif len(versions) > 1:
                message += f"(versions {','.join(versions)})"

            if idx + 1 < len(self._local_model_names):
                message += ", "
            else:
                message += " now available on disk"

        logger.info(message)

    def _update_models_tree(self) -> None:
        # get updated/validated paths/versions of the cloud models
        (
            model_names,
            versions,
            model_paths,
            sub_paths,
            timestamps,
            bucket_providers,
            bucket_names,
        ) = find_all_cloud_models(
            self._is_dir_used,
            self._models_dir,
            self._predictor_type,
            self._s3_paths,
            self._cloud_model_names,
        )

        # update models on the local disk if changes have been detected
        # a model is updated if its directory tree has changed, if it's not present or if it doesn't exist on the upstream
        with ThreadPoolExecutor(max_workers=5) as executor:
            futures = []
            for idx, (model_name, bucket_provider, bucket_name, bucket_sub_paths) in enumerate(
                zip(model_names, bucket_providers, bucket_names, sub_paths)
            ):
                futures += [
                    executor.submit(
                        self._refresh_model,
                        idx,
                        model_name,
                        model_paths[idx],
                        versions[model_name],
                        timestamps[idx],
                        bucket_sub_paths,
                        bucket_provider,
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
                    f.write("not-available")
            shutil.rmtree(os.path.join(self._download_dir, model_name))

        logger.debug(f"{self.__class__.__name__} cron heartbeat")

    def _refresh_model(
        self,
        idx: int,
        model_name: str,
        model_path: str,
        versions: List[str],
        timestamps: List[datetime.datetime],
        sub_paths: List[str],
        bucket_provider: str,
        bucket_name: str,
    ) -> None:

        if bucket_provider == "s3":
            client = S3(bucket_name)
        if bucket_provider == "gs":
            client = GCS(bucket_name)

        ondisk_model_path = os.path.join(self._download_dir, model_name)
        for version, model_ts in zip(versions, timestamps):

            # for the lock file
            resource = os.path.join(self._lock_dir, model_name + "-" + version + ".txt")

            # check if a model update is mandated
            update_model = False
            ondisk_model_version_path = os.path.join(ondisk_model_path, version)
            if os.path.exists(ondisk_model_version_path):
                local_paths = glob.glob(
                    os.path.join(ondisk_model_version_path, "**"), recursive=True
                )
                local_paths = util.remove_non_empty_directory_paths(local_paths)
                local_paths = [
                    os.path.relpath(local_path, ondisk_model_version_path)
                    for local_path in local_paths
                ]
                local_paths = [path for path in local_paths if not path.startswith("../")]

                cloud_model_version_path = os.path.join(model_path, version)
                cloud_paths = [
                    os.path.relpath(sub_path, cloud_model_version_path) for sub_path in sub_paths
                ]
                cloud_paths = [path for path in cloud_paths if not path.startswith("../")]
                cloud_paths = util.remove_non_empty_directory_paths(cloud_paths)

                # update if the paths don't match
                if set(local_paths) != set(cloud_paths):
                    update_model = True

                # update if the timestamp is newer
                with LockedFile(resource, "r", reader_lock=True) as f:
                    file_status = f.read()
                    if file_status == "" or file_status == "not-available":
                        raise WithBreak
                    current_model_ts = int(file_status.split(" ")[1])
                    if current_model_ts < int(model_ts.timestamp()):
                        update_model = True
            else:
                update_model = True

            if update_model:
                # download to a temp directory
                temp_dest = os.path.join(self._temp_dir, model_name, version)
                cloud_src = os.path.join(model_path, version)
                client.download_dir_contents(cloud_src, temp_dest)

                # validate the downloaded model
                model_contents = glob.glob(os.path.join(temp_dest, "**"), recursive=True)
                model_contents = util.remove_non_empty_directory_paths(model_contents)
                try:
                    validate_model_paths(model_contents, self._predictor_type, temp_dest)
                    passed_validation = True
                except CortexException:
                    passed_validation = False
                    shutil.rmtree(temp_dest)

                    if bucket_provider == "s3":
                        cloud_path = S3.construct_s3_path(bucket_name, cloud_src)
                    if bucket_provider == "gs":
                        cloud_path = GCS.construct_gcs_path(bucket_name, cloud_src)
                    logger.debug(
                        f"failed validating model {model_name} of version {version} found at {cloud_path} path"
                    )

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
            ondisk_model_versions = glob.glob(os.path.join(ondisk_model_path, "**"))
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
                        f.write("not-available")

            # remove the model directory if there are no models left
            if len(glob.glob(os.path.join(ondisk_model_path, "**"))) == 0:
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
                local_paths = glob.glob(
                    os.path.join(ondisk_model_version_path, "**"), recursive=True
                )
                local_paths = util.remove_non_empty_directory_paths(local_paths)
                local_paths = [
                    os.path.relpath(local_path, ondisk_model_version_path)
                    for local_path in local_paths
                ]
                local_paths = [path for path in local_paths if not path.startswith("../")]

                cloud_model_version_path = model_path
                cloud_paths = [
                    os.path.relpath(sub_path, cloud_model_version_path) for sub_path in sub_paths
                ]
                cloud_paths = [path for path in cloud_paths if not path.startswith("../")]
                cloud_paths = util.remove_non_empty_directory_paths(cloud_paths)

                # update if the paths don't match
                if set(local_paths) != set(cloud_paths):
                    update_model = True

                # update if the timestamp is newer
                with LockedFile(resource, "r", reader_lock=True) as f:
                    file_status = f.read()
                    if file_status == "" or file_status == "not-available":
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
            client.download_dir_contents(model_path, temp_dest)

            # validate the downloaded model
            model_contents = glob.glob(os.path.join(temp_dest, "**"), recursive=True)
            model_contents = util.remove_non_empty_directory_paths(model_contents)
            try:
                validate_model_paths(model_contents, self._predictor_type, temp_dest)
                passed_validation = True
            except CortexException:
                passed_validation = False
                shutil.rmtree(temp_dest)

                if bucket_provider == "s3":
                    cloud_path = S3.construct_s3_path(bucket_name, model_path)
                if bucket_provider == "gs":
                    cloud_path = GCS.construct_gcs_path(bucket_name, model_path)
                logger.debug(
                    f"failed validating model {model_name} of version {version} found at {cloud_path} path"
                )

            # move the model to its destination directory
            if passed_validation:
                with LockedFile(resource, "w+") as f:
                    if os.path.exists(ondisk_model_version_path):
                        shutil.rmtree(ondisk_model_version_path)
                    shutil.move(temp_dest, ondisk_model_version_path)
                    f.write("available " + str(model_ts))


class FileBasedModelsGC(AbstractLoopingThread):
    """
    GC for models that no longer exist on disk. To be used with FileBasedModelsTreeUpdater.

    There has to be a FileBasedModelsGC cron for each API process.

    This needs to run on the API process because the FileBasedModelsTreeUpdater process cannot
    unload the models from the API process' memory by itself. API process has to rely on this cron to do this periodically.

    This is for the case when the FileBasedModelsTreeUpdater process has removed models from disk and there are still models loaded into the API process' memory.
    """

    def __init__(
        self,
        interval: int,
        models: ModelsHolder,
        download_dir: str,
        lock_dir: str = "/run/cron",
    ):
        """
        Args:
            interval: How often to run the GC. Measured in seconds.
            download_dir: Path to where the models are stored.
            lock_dir: Path to where the resource locks are stored.
        """
        AbstractLoopingThread.__init__(self, interval, self._run_gc)

        self._models = models
        self._download_dir = download_dir
        self._lock_dir = lock_dir

    def _run_gc(self):
        on_disk_model_ids = find_ondisk_model_ids_with_lock(self._lock_dir)
        in_memory_model_ids = self._models.get_model_ids()

        logger.debug(f"{self.__class__.__name__} cron heartbeat")

        for in_memory_id in in_memory_model_ids:
            if in_memory_id in on_disk_model_ids:
                continue
            with LockedModel(self._models, "w", model_id=in_memory_id):
                if self._models.has_model_id(in_memory_id)[0] == "in-memory":
                    model_name, model_version = in_memory_id.rsplit("-", maxsplit=1)
                    logger.info(
                        f"removing model {model_name} of version {model_version} from memory as it's no longer present on disk/S3/GS (thread {td.get_ident()})"
                    )
                    self._models.remove_model_by_id(
                        in_memory_id, mem=True, disk=False, del_reference=True
                    )


def find_ondisk_models_with_lock(
    lock_dir: str, include_timestamps: bool = False
) -> Union[Dict[str, List[str]], Dict[str, Dict[str, Any]]]:
    """
    Returns all available models from the disk.
    To be used in conjunction with FileBasedModelsTreeUpdater/FileBasedModelsGC.

    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        lock_dir: Path to where the resource locks are stored.
        include_timestamps: Whether to include timestamps for each version of each model.

    Returns:
        Dictionary with available model names and their associated versions when include_timestamps is False.
        {
            "model-A": ["177", "245", "247"],
            "model-B": ["1"],
            ...
        }

        Dictionary with available model names and their associated versions/timestamps when include_timestamps is True.
        {
            "model-A": {
                "versions": ["177", "245", "247"],
                "timestamps": [1602198945, 1602198946, 1602198947]
            }
            "model-B": {
                "versions": ["1"],
                "timestamps": [1602198567]
            },
            ...
        }
    """
    models = {}

    for locked_file in get_locked_files(lock_dir):
        with LockedFile(os.path.join(lock_dir, locked_file), "r", reader_lock=True) as f:
            status = f.read()

        if status.startswith("available"):
            timestamp = int(status.split(" ")[1])
            _model_name, _model_version = os.path.splitext(locked_file)[0].rsplit("-", maxsplit=1)
            if _model_name not in models:
                if include_timestamps:
                    models[_model_name] = {"versions": [_model_version], "timestamps": [timestamp]}
                else:
                    models[_model_name] = [_model_version]
            else:
                if include_timestamps:
                    models[_model_name]["versions"] += [_model_version]
                    models[_model_name]["timestamps"] += [timestamp]
                else:
                    models[_model_name] += [_model_version]

    return models


def find_ondisk_model_ids_with_lock(lock_dir: str) -> List[str]:
    """
    Returns all available model IDs from the disk.
    To be used in conjunction with FileBasedModelsTreeUpdater/FileBasedModelsGC.

    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        lock_dir: Path to where the resource locks are stored.

    Returns:
        A list with all model IDs present on disk.
    """
    model_ids = []

    for locked_file in get_locked_files(lock_dir):
        with LockedFile(os.path.join(lock_dir, locked_file), "r", reader_lock=True) as f:
            status = f.read()

        if status.startswith("available"):
            model_id = os.path.splitext(locked_file)[0]
            model_ids.append(model_id)

    return model_ids


def find_ondisk_model_info(lock_dir: str, model_name: str) -> Tuple[List[str], List[int]]:
    """
    Returns all available versions/timestamps of a model from the disk.
    To be used in conjunction with FileBasedModelsTreeUpdater/FileBasedModelsGC.

    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        lock_dir: Path to where the resource locks are stored.
        model_name: Name of the model as specified in predictor:models:paths:name, _cortex_default when predictor:models:path is set or the discovered model names when predictor:models:dir is used.

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
    Monitors the cloud path(s)/dir (S3 or GS) and continuously updates the models on TFS.
    The model paths are validated - the bad paths are ignored.
    When a new model is found, it updates the tree, downloads it and loads it into memory - likewise when a model is removed.
    """

    def __init__(
        self,
        interval: int,
        api_spec: dict,
        tfs_model_dir: str,
        download_dir: str,
        address: Optional[str] = None,
        addresses: Optional[List[str]] = None,
        temp_dir: str = "/tmp/cron",
        lock_dir: str = "/run/cron",
    ):
        """
        Args:
            interval: How often to update the models tree. Measured in seconds.
            api_spec: Identical copy of pkg.type.spec.api.API.
            address: An address with the "host:port" format to where TFS is located.
            addresses: A list of addresses with the "host:port" format to where the TFS servers are located.
            tfs_model_dir: Path to where the models are stored within the TFS container.
            download_dir: Path to where the models are stored.
            temp_dir: Directory where models are temporarily stored.
            lock_dir: Directory in which model timestamps are stored.
        """

        if address and addresses:
            raise ValueError("address and addresses arguments cannot be passed in at the same time")
        if not address and not addresses:
            raise ValueError("must pass in at least one of the two arguments: address or addresses")

        mp.Process.__init__(self, daemon=True)

        self._interval = interval
        self._api_spec = api_spec
        self._tfs_model_dir = tfs_model_dir
        self._download_dir = download_dir
        self._temp_dir = temp_dir
        self._lock_dir = lock_dir

        if address:
            self._tfs_address = address
            self._tfs_addresses = None
        else:
            self._tfs_address = None
            self._tfs_addresses = addresses

        self._cloud_paths = []
        self._spec_models = get_models_from_api_spec(self._api_spec)
        self._cloud_model_names = self._spec_models.get_cloud_model_names()
        for model_name in self._cloud_model_names:
            self._cloud_paths.append(self._spec_models[model_name]["path"])

        if (
            self._api_spec["predictor"]["models"] is not None
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

        self._ran_once = mp.Event()
        self._event_stopper = mp.Event()
        self._stopped = mp.Event()

        # keeps an old record of the model timestamps
        self._old_ts_state = {}

    def run(self):
        """
        mp.Process-specific method.
        """

        init_sentry(tags=get_default_tags())
        if self._tfs_address:
            self._client = TensorFlowServingAPI(self._tfs_address)
        else:
            self._client = TensorFlowServingAPIClones(self._tfs_addresses)

        # wait until TFS is responsive
        while not self._client.is_tfs_accessible():
            self._reset_when_tfs_unresponsive()
            time.sleep(1.0)

        while not self._event_stopper.is_set():
            success = self._update_models()
            if success and not self._ran_once.is_set():
                self._ran_once.set()
            logger.debug(f"{self.__class__.__name__} cron heartbeat")
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

    def ran_once(self) -> bool:
        """
        Tells whether the TFS loader loop has run at least once.
        """

        return self._ran_once.is_set()

    def _update_models(self) -> bool:
        # get updated/validated paths/versions of the cloud models (S3 or GS)
        (
            model_names,
            versions,
            model_paths,
            sub_paths,
            timestamps,
            bucket_providers,
            bucket_names,
        ) = find_all_cloud_models(
            self._is_dir_used,
            self._models_dir,
            self._predictor_type,
            self._cloud_paths,
            self._cloud_model_names,
        )

        # update models on the local disk if changes have been detected
        # a model is updated if its directory tree has changed, if it's not present or if it doesn't exist on the upstream
        with ThreadPoolExecutor(max_workers=5) as executor:
            futures = []
            for idx, (model_name, bucket_provider, bucket_name, bucket_sub_paths) in enumerate(
                zip(model_names, bucket_providers, bucket_names, sub_paths)
            ):
                futures += [
                    executor.submit(
                        self._refresh_model,
                        idx,
                        model_name,
                        model_paths[idx],
                        versions[model_name],
                        timestamps[idx],
                        bucket_sub_paths,
                        bucket_provider,
                        bucket_name,
                    )
                ]
            [future.result() for future in futures]

        # remove models that no longer appear in model_names
        for model_name, model_versions in find_ondisk_models(self._download_dir).items():
            if model_name in model_names:
                continue
            for ondisk_version in model_versions:
                ondisk_model_version_path = os.path.join(
                    self._download_dir, model_name, ondisk_version
                )
                shutil.rmtree(ondisk_model_version_path)
            shutil.rmtree(os.path.join(self._download_dir, model_name))
            self._client.remove_models([model_name], [model_versions])

        # check tfs connection
        if not self._client.is_tfs_accessible():
            self._reset_when_tfs_unresponsive()
            return False

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
                except grpc.RpcError as err:
                    if err.code() == grpc.StatusCode.UNAVAILABLE:
                        logger.warning(
                            "TFS server unresponsive after trying to load model '{}' of version '{}': {}".format(
                                model_name, model_version, str(err)
                            )
                        )
                    self._reset_when_tfs_unresponsive()
                    return False

        # # update TFS models
        current_ts_state = {}
        for model_name, model_versions in ondisk_models.items():
            try:
                ts = self._update_tfs_model(
                    model_name, model_versions, timestamps, model_names, versions
                )
            except grpc.RpcError:
                return False
            current_ts_state = {**current_ts_state, **ts}

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
        with open(resource, "w") as f:
            json.dump(self._old_ts_state, f, indent=2)

        # save model stats for TFS to disk
        resource = os.path.join(self._lock_dir, "models_tfs.json")
        with open(resource, "w") as f:
            json.dump(self._client.models, f, indent=2)

        return True

    def _refresh_model(
        self,
        idx: int,
        model_name: str,
        model_path: str,
        versions: List[str],
        timestamps: List[datetime.datetime],
        sub_paths: List[str],
        bucket_provider: str,
        bucket_name: str,
    ) -> None:

        if bucket_provider == "s3":
            client = S3(bucket_name)
        if bucket_provider == "gs":
            client = GCS(bucket_name)

        ondisk_model_path = os.path.join(self._download_dir, model_name)
        for version, model_ts in zip(versions, timestamps):

            # check if a model update is mandated
            update_model = False
            ondisk_model_version_path = os.path.join(ondisk_model_path, version)
            if os.path.exists(ondisk_model_version_path):
                local_paths = glob.glob(
                    os.path.join(ondisk_model_version_path, "**"), recursive=True
                )
                local_paths = util.remove_non_empty_directory_paths(local_paths)
                local_paths = [
                    os.path.relpath(local_path, ondisk_model_version_path)
                    for local_path in local_paths
                ]
                local_paths = [path for path in local_paths if not path.startswith("../")]

                cloud_model_version_path = os.path.join(model_path, version)
                cloud_paths = [
                    os.path.relpath(sub_path, cloud_model_version_path) for sub_path in sub_paths
                ]
                cloud_paths = [path for path in cloud_paths if not path.startswith("../")]
                cloud_paths = util.remove_non_empty_directory_paths(cloud_paths)

                if set(local_paths) != set(cloud_paths):
                    update_model = True

                model_id = f"{model_name}-{version}"
                if self._is_this_a_newer_model_id(model_id, int(model_ts.timestamp())):
                    update_model = True
            else:
                update_model = True

            if update_model:
                # download to a temp directory
                temp_dest = os.path.join(self._temp_dir, model_name, version)
                cloud_src = os.path.join(model_path, version)
                client.download_dir_contents(cloud_src, temp_dest)

                # validate the downloaded model
                model_contents = glob.glob(os.path.join(temp_dest, "**"), recursive=True)
                model_contents = util.remove_non_empty_directory_paths(model_contents)
                try:
                    validate_model_paths(model_contents, self._predictor_type, temp_dest)
                    passed_validation = True
                except CortexException:
                    passed_validation = False
                    shutil.rmtree(temp_dest)

                    if bucket_provider == "s3":
                        cloud_path = S3.construct_s3_path(bucket_name, model_path)
                    if bucket_provider == "gs":
                        cloud_path = GCS.construct_gcs_path(bucket_name, model_path)
                    logger.debug(
                        f"failed validating model {model_name} of version {version} found at {cloud_path} path"
                    )

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
            ondisk_model_versions = glob.glob(os.path.join(ondisk_model_path, "**"))
            ondisk_model_versions = [
                os.path.relpath(path, ondisk_model_path) for path in ondisk_model_versions
            ]
            for ondisk_version in ondisk_model_versions:
                if ondisk_version not in versions and (ondisk_version != "1" or len(versions) > 0):
                    ondisk_model_version_path = os.path.join(ondisk_model_path, ondisk_version)
                    shutil.rmtree(ondisk_model_version_path)

            if len(glob.glob(os.path.join(ondisk_model_path, "**"))) == 0:
                shutil.rmtree(ondisk_model_path)

        # if it's a non-versioned model ModelVersion.NOT_PROVIDED
        if len(versions) == 0 and len(sub_paths) > 0:

            model_ts = timestamps[0]

            # check if a model update is mandated
            update_model = False
            ondisk_model_version_path = os.path.join(ondisk_model_path, "1")
            if os.path.exists(ondisk_model_version_path):
                local_paths = glob.glob(
                    os.path.join(ondisk_model_version_path, "**"), recursive=True
                )
                local_paths = util.remove_non_empty_directory_paths(local_paths)
                local_paths = [
                    os.path.relpath(local_path, ondisk_model_version_path)
                    for local_path in local_paths
                ]
                local_paths = [path for path in local_paths if not path.startswith("../")]

                cloud_model_version_path = model_path
                cloud_paths = [
                    os.path.relpath(sub_path, cloud_model_version_path) for sub_path in sub_paths
                ]
                cloud_paths = [path for path in cloud_paths if not path.startswith("../")]
                cloud_paths = util.remove_non_empty_directory_paths(cloud_paths)

                # update if the paths don't match
                if set(local_paths) != set(cloud_paths):
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
            client.download_dir_contents(model_path, temp_dest)

            # validate the downloaded model
            model_contents = glob.glob(os.path.join(temp_dest, "**"), recursive=True)
            model_contents = util.remove_non_empty_directory_paths(model_contents)
            try:
                validate_model_paths(model_contents, self._predictor_type, temp_dest)
                passed_validation = True
            except CortexException:
                passed_validation = False
                shutil.rmtree(temp_dest)

                if bucket_provider == "s3":
                    cloud_path = S3.construct_s3_path(bucket_name, model_path)
                if bucket_provider == "gs":
                    cloud_path = GCS.construct_gcs_path(bucket_name, model_path)
                logger.debug(
                    f"failed validating model {model_name} of version {version} found at {cloud_path} path"
                )

            # move the model to its destination directory
            if passed_validation:
                if os.path.exists(ondisk_model_version_path):
                    shutil.rmtree(ondisk_model_version_path)
                shutil.move(temp_dest, ondisk_model_version_path)

    def _update_tfs_model(
        self,
        model_name: str,
        model_versions: List[str],
        _cloud_timestamps: List[List[datetime.datetime]],
        _cloud_model_names: List[str],
        _cloud_versions: Dict[str, List[str]],
    ) -> Optional[dict]:
        """
        Compares the existing models from TFS with those present on disk.
        Does the loading/unloading/reloading of models.

        From the _cloud_timestamps, _cloud_model_names, _cloud_versions params, only the fields of the respective model name are used.
        """

        # to prevent overwriting mistakes
        cloud_timestamps = copy.deepcopy(_cloud_timestamps)
        cloud_model_names = copy.deepcopy(_cloud_model_names)
        cloud_versions = copy.deepcopy(_cloud_versions)

        current_ts_state = {}

        # get the right order of model versions with respect to the model ts order
        model_timestamps = cloud_timestamps[cloud_model_names.index(model_name)]
        filtered_model_versions = []
        if len(cloud_versions[model_name]) == 0:
            filtered_model_versions = ["1"] * len(model_timestamps)
        else:
            for idx in range(len(model_timestamps)):
                if cloud_versions[model_name][idx] in model_versions:
                    filtered_model_versions.append(cloud_versions[model_name][idx])

        for model_version, model_ts in zip(filtered_model_versions, model_timestamps):
            model_ts = int(model_ts.timestamp())

            # remove outdated model
            model_id = f"{model_name}-{model_version}"
            is_model_outdated = False
            first_time_load = False
            if model_id in self._old_ts_state and self._old_ts_state[model_id] != model_ts:
                try:
                    self._client.remove_single_model(model_name, model_version)
                except grpc.RpcError as err:
                    if err.code() == grpc.StatusCode.UNAVAILABLE:
                        logger.warning(
                            "TFS server unresponsive after trying to unload model '{}' of version '{}': {}".format(
                                model_name, model_version, str(err)
                            )
                        )
                    logger.warning("waiting for tensorflow serving")
                    raise
                is_model_outdated = True
            elif model_id not in self._old_ts_state:
                first_time_load = True

            if not is_model_outdated and not first_time_load:
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
                except grpc.RpcError as err:
                    if err.code() == grpc.StatusCode.UNAVAILABLE:
                        logger.warning(
                            "TFS server unresponsive after trying to load model '{}' of version '{}': {}".format(
                                model_name, model_version, str(err)
                            )
                        )
                    self._reset_when_tfs_unresponsive()
                    raise

                is_model_outdated = False
                first_time_load = False

            # save timestamp of loaded model
            current_ts_state[model_id] = model_ts
            if is_model_outdated:
                logger.info(
                    "model '{}' of version '{}' has been reloaded".format(model_name, model_version)
                )
            elif first_time_load:
                logger.info(
                    "model '{}' of version '{}' has been loaded".format(model_name, model_version)
                )

        return current_ts_state

    def _is_this_a_newer_model_id(self, model_id: str, timestamp: int) -> bool:
        return model_id in self._old_ts_state and self._old_ts_state[model_id] < timestamp

    def _determine_model_signature_key(self, model_name: str) -> Optional[str]:
        if self._models_dir:
            signature_key = self._api_spec["predictor"]["models"]["signature_key"]
        else:
            signature_key = self._spec_models[model_name]["signature_key"]

        return signature_key

    def _reset_when_tfs_unresponsive(self):
        logger.warning("waiting for tensorflow serving")

        if self._tfs_address:
            self._client = TensorFlowServingAPI(self._tfs_address)
        else:
            self._client = TensorFlowServingAPIClones(self._tfs_addresses)

        resource = os.path.join(self._lock_dir, "models_tfs.json")
        with open(resource, "w") as f:
            json.dump(self._client.models, f, indent=2)


class TFSAPIServingThreadUpdater(AbstractLoopingThread):
    """
    When live reloading and the TensorFlow predictor are used, the serving container
    needs to have a way of accessing the models' metadata which is generated using the TFSModelLoader cron.

    This cron runs on each serving process and periodically reads the exported metadata from the TFSModelLoader cron.
    This is then fed into each serving process.
    """

    def __init__(
        self,
        interval: int,
        client: TensorFlowServingAPI,
        lock_dir: str = "/run/cron",
    ):
        AbstractLoopingThread.__init__(self, interval, self._run_tfs)

        self._client = client
        self._lock_dir = lock_dir

    def _run_tfs(self) -> None:
        resource_models = os.path.join(self._lock_dir, "models_tfs.json")

        try:
            with open(resource_models, "r") as f:
                models = json.load(f)
        except Exception:
            return

        resource_ts = os.path.join(self._lock_dir, "model_timestamps.json")
        try:
            with open(resource_ts, "r") as f:
                timestamps = json.load(f)
        except Exception:
            return

        non_intersecting_model_ids = set(models.keys()).symmetric_difference(timestamps.keys())
        for non_intersecting_model_id in non_intersecting_model_ids:
            if non_intersecting_model_id in models:
                del models[non_intersecting_model_id]
            if non_intersecting_model_id in timestamps:
                del timestamps[non_intersecting_model_id]

        for model_id in timestamps.keys():
            models[model_id]["timestamp"] = timestamps[model_id]

        self._client.models = models


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

        AbstractLoopingThread.__init__(self, interval, self._run_gc)

        self._api_spec = api_spec
        self._models = models
        self._tree = tree

        self._spec_models = get_models_from_api_spec(self._api_spec)

        # only required for ONNX file paths
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

        self._event_stopper = td.Event()
        self._stopped = False

    def _run_gc(self) -> None:

        # are there any models to collect (aka remove) from cache
        with LockedGlobalModelsGC(self._models, "r"):
            collectible, _, _ = self._models.garbage_collect(
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

            _, memory_evicted_model_ids, disk_evicted_model_ids = self._models.garbage_collect(
                exclude_disk_model_ids=self._local_model_ids
            )

        # otherwise, grab exclusive access to all models with exclusive access preference
        # and remove excess models from cache
        if not acquired:
            self._models.set_global_preference_policy("w")
            with LockedGlobalModelsGC(self._models, "w"):
                _, memory_evicted_model_ids, disk_evicted_model_ids = self._models.garbage_collect(
                    exclude_disk_model_ids=self._local_model_ids
                )
            self._models.set_global_preference_policy("r")

        memory_evicted_models = ids_to_models(memory_evicted_model_ids)
        disk_evicted_models = ids_to_models(disk_evicted_model_ids)

        self._log_removed_models(memory_evicted_models, memory=True)
        self._log_removed_models(disk_evicted_models, disk=True)

        self._remove_stale_models()

    def _remove_stale_models(self) -> None:
        """
        Remove models that exist locally in-memory and on-disk that no longer appear on the cloud upstream (S3 or GS).
        """

        # get available upstream S3 model IDs
        cloud_model_names = self._tree.get_model_names()
        cloud_model_versions = [
            self._tree.model_info(model_name)["versions"] for model_name in cloud_model_names
        ]
        cloud_model_ids = []
        for model_name, model_versions in zip(cloud_model_names, cloud_model_versions):
            if len(model_versions) == 0:
                continue
            for model_version in model_versions:
                cloud_model_ids.append(f"{model_name}-{model_version}")

        # get model IDs loaded into memory or on disk.
        with LockedGlobalModelsGC(self._models, "r"):
            present_model_ids = self._models.get_model_ids()

        # exclude local models from removal
        present_model_ids = list(set(present_model_ids) - set(self._local_model_ids))

        # remove models that don't exist in the S3 upstream
        ghost_model_ids = list(set(present_model_ids) - set(cloud_model_ids))
        for model_id in ghost_model_ids:
            model_name, model_version = model_id.rsplit("-", maxsplit=1)
            with LockedModel(self._models, "w", model_name, model_version):
                status, ts = self._models.has_model(model_name, model_version)
                if status == "in-memory":
                    logger.info(
                        f"unloading stale model {model_name} of version {model_version} using the garbage collector"
                    )
                    self._models.unload_model(model_name, model_version)
                if status in ["in-memory", "on-disk"]:
                    logger.info(
                        f"removing stale model {model_name} of version {model_version} using the garbage collector"
                    )
                    self._models.remove_model(model_name, model_version)

    def _log_removed_models(
        self, models: Dict[str, List[str]], memory: bool = False, disk: bool = False
    ) -> None:
        """
        Log the removed models from disk/memory.
        """

        if len(models) == 0:
            return None

        if len(models) > 1:
            message = "models "
        else:
            message = "model "

        for idx, (model_name, versions) in enumerate(models.items()):
            message += f"{model_name} "
            if len(versions) == 1:
                message += f"(version {versions[0]})"
            else:
                message += f"(versions {','.join(versions)})"
            if idx + 1 < len(models):
                message += ", "
            else:
                if memory:
                    message += " removed from the memory cache using the garbage collector"
                if disk:
                    message += " removed from the disk cache using the garbage collector"

        logger.info(message)


class ModelTreeUpdater(AbstractLoopingThread):
    """
    Model tree updater. Updates a local representation of all available models from the cloud upstreams (S3 or GS).
    """

    def __init__(self, interval: int, api_spec: dict, tree: ModelsTree, ondisk_models_dir: str):
        """
        Args:
            interval: How often to update the models tree. Measured in seconds.
            api_spec: Identical copy of pkg.type.spec.api.API.
            tree: Model tree representation of the available models on the cloud upstream (S3 or GS).
            ondisk_models_dir: Where the models are stored on disk. Necessary when local models are used.
        """

        AbstractLoopingThread.__init__(self, interval, self._update_models_tree)

        self._api_spec = api_spec
        self._tree = tree
        self._ondisk_models_dir = ondisk_models_dir

        self._cloud_paths = []
        self._spec_models = get_models_from_api_spec(self._api_spec)
        self._cloud_model_names = self._spec_models.get_cloud_model_names()
        for model_name in self._cloud_model_names:
            self._cloud_paths.append(self._spec_models[model_name]["path"])

        self._predictor_type = predictor_type_from_api_spec(self._api_spec)

        if (
            self._predictor_type == PythonPredictorType
            and self._api_spec["predictor"]["multi_model_reloading"]
        ):
            models = self._api_spec["predictor"]["multi_model_reloading"]
        elif self._predictor_type != PythonPredictorType:
            models = self._api_spec["predictor"]["models"]
        else:
            models = None

        if models is None:
            raise CortexException("no specified model")

        if models and models["dir"] is not None:
            self._is_dir_used = True
            self._models_dir = models["dir"]
        else:
            self._is_dir_used = False
            self._models_dir = None

        # only required for ONNX file paths
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
                ondisk_paths = glob.glob(
                    os.path.join(ondisk_model_version_path, "**"), recursive=True
                )
                ondisk_paths = util.remove_non_empty_directory_paths(ondisk_paths)
                # removable is set to false to prevent the local models from being removed
                self._tree.update_model(
                    provider="",
                    bucket="",
                    model_name=model_name,
                    model_version=model_version,
                    model_path=ondisk_model_version_path,
                    sub_paths=ondisk_paths,
                    timestamp=timestamp_utc,
                    removable=False,
                )

            for model_version in model["versions"]:
                ondisk_model_version_path = os.path.join(
                    self._ondisk_models_dir, model_name, model_version
                )
                ondisk_paths = glob.glob(
                    os.path.join(ondisk_model_version_path, "**"), recursive=True
                )
                ondisk_paths = util.remove_non_empty_directory_paths(ondisk_paths)
                # removable is set to false to prevent the local models from being removed
                self._tree.update_model(
                    provider="",
                    bucket="",
                    model_name=model_name,
                    model_version=model_version,
                    model_path=ondisk_model_version_path,
                    sub_paths=ondisk_paths,
                    timestamp=timestamp_utc,
                    removable=False,
                )

    def _update_models_tree(self) -> None:
        # get updated/validated paths/versions of the cloud models (S3 or GS)
        (
            model_names,
            versions,
            model_paths,
            sub_paths,
            timestamps,
            bucket_providers,
            bucket_names,
        ) = find_all_cloud_models(
            self._is_dir_used,
            self._models_dir,
            self._predictor_type,
            self._cloud_paths,
            self._cloud_model_names,
        )

        # update model tree
        self._tree.update_models(
            model_names,
            versions,
            model_paths,
            sub_paths,
            timestamps,
            bucket_providers,
            bucket_names,
        )

        logger.debug(f"{self.__class__.__name__} cron heartbeat")
