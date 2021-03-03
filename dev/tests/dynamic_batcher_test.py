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

# Dependencies: cortex_internal, pystuck
# When this is running, to inspect the stuck threads upon exiting (by pressing CTRL-C), run pystuck (as a CLI) in a different window

import time
import itertools
import threading as td
import signal
import sys

import pystuck
from cortex_internal.lib.api import batching

pystuck.run_server()


class Predictor:
    def predict(self, payload):
        print("received payload:", payload)
        time.sleep(0.2)
        return payload


db = batching.DynamicBatcher(Predictor(), max_batch_size=32, batch_interval=0.1)
counter = itertools.count(1)
event = td.Event()
global_list = []
running_threads = []


def submitter():
    while not event.is_set():
        global_list.append(db.predict(payload=next(counter)))
        time.sleep(0.1)


def signal_handler(sig, frame):
    event.set()
    print("ctrl-c has been pressed; exiting ...")
    print(
        f"global_list(global_list+1)/2={int(len(global_list) * (len(global_list) + 1) / 2)} || sum(global_list)={sum(global_list)}"
    )
    for thread in running_threads:
        print("joining on", thread.getName())
        thread.join()
    sys.exit(0)


signal.signal(signal.SIGINT, signal_handler)

for i in range(128):
    thread = td.Thread(target=submitter)
    thread.start()
    running_threads.append(thread)
