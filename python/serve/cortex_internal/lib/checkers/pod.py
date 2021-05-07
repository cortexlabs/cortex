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
import stat
import time

from cortex_internal import consts


def neuron_socket_exists():
    if not os.path.exists(consts.INFERENTIA_NEURON_SOCKET):
        return False
    else:
        mode = os.stat(consts.INFERENTIA_NEURON_SOCKET)
        return stat.S_ISSOCK(mode.st_mode)


def wait_neuron_rtd():
    while not neuron_socket_exists():
        time.sleep(0.1)
