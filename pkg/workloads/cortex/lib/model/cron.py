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

import threading as td
import multiprocessing as mp
import time


class SimpleModelMonitor(mp.Process):
    """
    Responsible for monitoring the S3 path(s)/dir and continuously update the tree.
    The model paths are validated - the bad paths are ignored.
    When a new model is found, it updates the tree and downloads it - likewise when a model is removed.
    """

    def __init__(self, interval: int, paths: list(str), is_path_top_dir: bool, **kwargs):
        """
        Args:
            interval (int): How often to update the models tree. Measured in seconds.
            paths (list(str)): The model paths as passed in cortex.yaml. Can be predictor:model_path, predictor:models:paths:model_path or predictor:models:dir.
            is_path_top_dir (str): Whether the models have been specified using predictor:models:dir or not.
        """
        mp.Process.__init__(self, **kwargs)
        self._interval = interval
        self._paths = paths
        self.is_path_top_dir = is_path_to_dir
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
        pass


class CachedModelMonitor(td.Thread):
    """
    Responsible for monitoring the S3 path(s)/dir and continuously update the tree.
    The model paths are validated - the bad paths are ignored.
    When a new model is found, it updates the tree - likewise when a model is removed.

    Also has access to the shared models of all other threads and so if the cache size thresholds, then evict the LRU.
    Does the same for the disk cache size.
    """

    def __init__(self, interval: int, **kwargs):
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
