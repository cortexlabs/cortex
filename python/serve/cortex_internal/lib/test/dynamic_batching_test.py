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


import itertools
import threading as td
import time

from cortex_internal.lib.api.utils import DynamicBatcher


class Handler:
    def handle_post(self, payload):
        time.sleep(0.2)
        return payload


def test_dynamic_batching_while_hitting_max_batch_size():
    max_batch_size = 32
    dynamic_batcher = DynamicBatcher(
        Handler(),
        method_name="handle_post",
        max_batch_size=max_batch_size,
        batch_interval=0.1,
        test_mode=True,
    )
    counter = itertools.count(1)
    event = td.Event()
    global_list = []

    def submitter():
        while not event.is_set():
            global_list.append(dynamic_batcher.process(payload=next(counter)))
            time.sleep(0.1)

    running_threads = []
    for _ in range(128):
        thread = td.Thread(target=submitter, daemon=True)
        thread.start()
        running_threads.append(thread)

    time.sleep(60)
    event.set()

    # if this fails, then the submitter threads are getting stuck
    for thread in running_threads:
        thread.join(3.0)
        if thread.is_alive():
            raise TimeoutError("thread", thread.getName(), "got stuck")

    sum1 = int(len(global_list) * (len(global_list) + 1) / 2)
    sum2 = sum(global_list)
    assert sum1 == sum2

    # get the last 80% of batch lengths
    # we ignore the first 20% because it may take some time for all threads to start making requests
    batch_lengths = dynamic_batcher._test_batch_lengths
    batch_lengths = batch_lengths[int(len(batch_lengths) * 0.2) :]

    # verify that the batch size is always equal to the max batch size
    assert len(set(batch_lengths)) == 1
    assert max_batch_size in batch_lengths


def test_dynamic_batching_while_hitting_max_interval():
    max_batch_size = 32
    dynamic_batcher = DynamicBatcher(
        Handler(),
        method_name="handle_post",
        max_batch_size=max_batch_size,
        batch_interval=1.0,
        test_mode=True,
    )
    counter = itertools.count(1)
    event = td.Event()
    global_list = []

    def submitter():
        while not event.is_set():
            global_list.append(dynamic_batcher.process(payload=next(counter)))
            time.sleep(0.1)

    running_threads = []
    for _ in range(2):
        thread = td.Thread(target=submitter, daemon=True)
        thread.start()
        running_threads.append(thread)

    time.sleep(30)
    event.set()

    # if this fails, then the submitter threads are getting stuck
    for thread in running_threads:
        thread.join(3.0)
        if thread.is_alive():
            raise TimeoutError("thread", thread.getName(), "got stuck")

    sum1 = int(len(global_list) * (len(global_list) + 1) / 2)
    sum2 = sum(global_list)
    assert sum1 == sum2

    # get the last 80% of batch lengths
    # we ignore the first 20% because it may take some time for all threads to start making requests
    batch_lengths = dynamic_batcher._test_batch_lengths
    batch_lengths = batch_lengths[int(len(batch_lengths) * 0.2) :]

    # verify that the batch size is always equal to the number of running threads
    assert len(set(batch_lengths)) == 1
    assert len(running_threads) in batch_lengths
