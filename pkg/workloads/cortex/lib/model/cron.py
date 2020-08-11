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
from cortex.lib.storage import S3, LocalStorage, FileLock
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


class SimpleModelMonitor(mp.Process):
    """
    Responsible for monitoring the S3 path(s)/dir and continuously update the tree.
    The model paths are validated - the bad paths are ignored.
    When a new model is found, it updates the tree and downloads it - likewise when a model is removed.
    """

    def __init__(
        self, interval: int, api_spec: dict, download_dir: str, lock_dir: str = "/run/cron"
    ):
        """
        Args:
            interval (int): How often to update the models tree. Measured in seconds.
            kwargs: Named parameters.
        """

        mp.Process.__init__(self)
        self._interval = interval
        self._api_spec = api_spec
        self._download_dir = download_dir
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
        while not self._event_stopper.is_set():
            self._update_models_tree()
            time.sleep(self._interval)
        self._stopped.set()

    def stop(self, blocking: bool = False):
        self._event_stopper.set()
        if blocking:
            self.join()

    def join(self):
        while not self._stopped.is_set():
            time.sleep(0.001)

    def _update_models_tree(self):
        if self._is_dir_used:
            bucket_name, models_path = S3.deconstruct_s3_path(self._models_dir)
            s3_client = S3(bucket_name, client_config={})
            sub_paths = s3_client.search(models_path)
            model_paths = validate_s3_models_dir_paths(sub_paths, self._predictor_type, models_path)
            model_names = [os.path.basename(model_path) for model_path in model_paths]

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
                        validate_s3_model_paths(sub_paths, self._predictor_type, model_path)
                        model_paths.append(model_path)
                        model_names.append(self._model_names[idx])
                    except CortexException:
                        continue

        if self._is_dir_used:
            model_names = list(set(model_names).difference(self._local_model_names))
            model_paths = [
                model_path
                for model_path in models_path
                if os.path.basename(model_path) in model_names
            ]

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
        while not self._event_stopper.is_set():
            self._update_models_tree()
            time.sleep(self._interval)
        self._stopped = True

    def stop(self, blocking: bool = False):
        self._event_stopper.set()
        if blocking:
            self.join()

    def join(self):
        while not self._stopped:
            time.sleep(0.001)

    def _update_models_tree(self):
        pass
