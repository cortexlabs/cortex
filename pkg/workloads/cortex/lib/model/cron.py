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


class SimpleModelMonitor(mp.Process):
    """
    Responsible for monitoring the S3 path(s)/dir and continuously update the tree.
    The model paths are validated - the bad paths are ignored.
    When a new model is found, it updates the tree and downloads it - likewise when a model is removed.
    """

    pass


class CachedModelMonitor(td.Thread):
    """
    Responsible for monitoring the S3 path(s)/dir and continuously update the tree.
    The model paths are validated - the bad paths are ignored.
    When a new model is found, it updates the tree - likewise when a model is removed.

    Also has access to the shared models of all other threads and so if the cache size thresholds, then evict the LRU.
    Does the same for the disk cache size.
    """

    pass
