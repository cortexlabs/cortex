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

from typing import Dict, List, Tuple, Any

from cortex.lib import util
from cortex.lib.log import cx_logger
from cortex.lib.storage import S3, LocalStorage, LockedFile
from cortex.lib.exceptions import CortexException
from cortex.lib.model import (
    PythonPredictorType,
    TensorFlowPredictorType,
    TensorFlowNeuronPredictorType,
    ONNXPredictorType,
    validate_s3_models_dir_paths,
    validate_s3_model_paths,
)

import os
import threading as td
import multiprocessing as mp
import time
import glob
import shutil


class SimpleModelMonitor(mp.Process):
    """
    Responsible for monitoring the S3 path(s)/dir and continuously update the tree.
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

        self._paths = []
        self._model_names = []
        self._local_model_names = []
        for curated_model in self._api_spec["curated_model_resources"]:
            if curated_model["s3_path"]:
                self._paths.append(curated_model["model_path"])
                self._model_names.append(curated_model["name"])
            else:
                self._local_model_names.append(curated_model["name"])
        self._curated_models = self._api_spec["curated_model_resources"]

        if (
            self._api_spec["predictor"]["model_path"] is None
            and self._api_spec["predictor"]["models"]["dir"] is not None
        ):
            self._is_dir_used = True
            self._models_dir = self._api_spec["predictor"]["models"]["dir"]
        else:
            self._is_dir_used = False

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

    def _refresh_model(
        self,
        idx: int,
        model_name: str,
        versions: dict,
        model_paths: List[str],
        sub_paths: List[str],
    ):
        ondisk_model_path = os.path.join(self._download_dir, model_name)
        for version in versions[model_name]:

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
                        os.path.makedirs(ondisk_model_version)
                        shutil.copytree(temp_dest, ondisk_model_version)
                        f.write("available")

                    # cleanup
                    shutil.rmtree(temp_dest)

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

        # if it's a model that came without version (i.e. like it's the case for ONNXPredictorType)
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
                    os.path.makedirs(ondisk_model_version)
                    shutil.copytree(temp_dest, ondisk_model_version)
                    f.write("available")

                # cleanup
                shutil.rmtree(temp_dest)

    def _update_models_tree(self):
        # validate models stored in S3 that were specified with predictor:models:dir field
        if self._is_dir_used:
            bucket_name, models_path = S3.deconstruct_s3_path(self._models_dir)
            s3_client = S3(bucket_name, client_config={})
            sub_paths = s3_client.search(models_path)
            model_paths = validate_models_dir_paths(sub_paths, self._predictor_type, models_path)
            model_names = [os.path.basename(model_path) for model_path in model_paths]

            model_names = list(set(model_names).difference(self._local_model_names))
            model_paths = [
                model_path
                for model_path in models_path
                if os.path.basename(model_path) in model_names
            ]

        # validate models stored in S3 that were specified with predictor:models:paths field
        if not self._is_dir_used:
            sub_paths = []
            model_paths = []
            model_names = []
            for idx, path in enumerate(self._paths):
                if S3.is_valid_s3_path(path):
                    bucket_name, model_path = S3.deconstruct_s3_path(path)
                    s3_client = S3(bucket_name, client_config={})
                    sub_paths += s3_client.search(model_path)
                    try:
                        validate_model_paths(sub_paths, self._predictor_type, model_path)
                        model_paths.append(model_path)
                        model_names.append(self._model_names[idx])
                    except CortexException:
                        continue

        # determine the detected versions for each model
        # if the predictor type is ONNXPredictorType and the model contains a single *.onnx file,
        # then leave the version list empty
        versions = {}
        for model_path, model_name in zip(model_paths, model_names):
            model_sub_paths = [os.path.relpath(sub_path, model_path) for sub_path in sub_paths]
            model_sub_paths = [path for path in model_sub_paths if not path.startswith("../")]
            # make isnumeric verification because for ONNX models, the model path can be the actual ONNX file
            model_versions = [version for version in model_sub_paths if version.isnumeric()]
            versions[model_name] = model_versions

        # model_names - a list with the names of the models (i.e. bert, gpt-2, etc) and they are unique
        # versions - a dictionary with the keys representing the model names and the values being lists of versions that each model has.
        #   For ONNX model paths that are not versioned, the list will be empty
        # model_paths - a list with the prefix of each model
        # sub_paths - a list of filepaths for each file of each model all grouped into a single list

        # update models on the local disk if changes have been detected
        # a model is updated if its directory tree has changed, if it's not present or if it doesn't exist on the upstream
        for idx, model_name in enumerate(model_names):
            self._refresh_model(idx, model_name, versions, model_paths, sub_paths)

        print("model names", model_names)
        print("model paths", model_paths)
        print("sub paths", sub_paths)


class CachedModelMonitor(td.Thread):
    """
    Responsible for monitoring the S3 path(s)/dir and continuously update the tree.
    The model paths are validated - the bad paths are ignored.
    When a new model is found, it updates the tree - likewise when a model is removed.

    Also has access to the shared models of all other threads and so if the cache size thresholds, then evict the LRU.
    Does the same for the disk cache size.
    """

    def __init__(self, interval: int, **kwargs):
        """
        Args:
            interval (int): How often to update the models tree. Measured in seconds.
            kwargs: Named parameters.
        """

        mp.Thread.__init__(self, **kwargs)
        self._interval = interval
        self._event_stopper = thread.Event()
        self._stopped = False

    def run(self):
        """
        mp.Process-specific method.
        """

        while not self._event_stopper.is_set():
            self._update_models_tree()
            time.sleep(self._interval)
        self._stopped = True

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

        while not self._stopped:
            time.sleep(0.001)

    def _update_models_tree(self):
        pass


def find_ondisk_models(lock_dir: str) -> List[str]:
    """
    Returns all available models from the disk.

    This function should never be used for determining whether a model has to be loaded or not.
    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        lock_dir: Path to where the resource locks are stored.

    Returns:
        List with the available models from disk. Just the model, no versions.
    """

    files = [os.path.basename(file) for file in os.listdir(lock_dir)]
    locks = [f for f in files if f.endswith(".lock")]
    models = []

    for lock in locks:
        _model_name, _ = os.path.splitext(lock).split("-")
        with LockedFile(os.path.join(lock_dir, lock), reader_lock=True) as f:
            status = f.read()
        if status == "available":
            if _model_name not in models:
                models.append(_model_name)

    return models


def find_ondisk_model_versions(lock_dir: str, model_name: str) -> List[str]:
    """
    Returns all available versions of a model from the disk.

    This function should never be used for determining whether a model has to be loaded or not.
    Can be used for Python/TensorFlow/ONNX clients.

    Args:
        lock_dir: Path to where the resource locks are stored.
        model_name: Name of the model as specified in predictor:models:paths:name, _cortex_default when predictor:model_path is set or the discovered model names when predictor:models:dir is used.

    Returns:
        List with the available versions. Empty when the model is not available.
    """

    files = [os.path.basename(file) for file in os.listdir(lock_dir)]
    locks = [f for f in files if f.endswith(".lock")]
    versions = []

    for lock in locks:
        _model_name, version = os.path.splitext(lock).split("-")
        if _model_name == model_name:
            with LockedFile(os.path.join(lock_dir, lock), reader_lock=True) as f:
                status = f.read()
            if status == "available":
                versions.append(version)

    return versions
