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

from signal import signal, SIGINT, SIGTERM

from cortex_internal.lib.log import logger as log


class SignalHandler:
    def __init__(self):
        self.__received_signal = False
        signal(SIGINT, self._signal_handler)
        signal(SIGTERM, self._signal_handler)

    def _signal_handler(self, sys_signal, _):
        log.info(f"handling signal {sys_signal}, exiting gracefully")
        self.__received_signal = True

    def received_signal(self):
        return self.__received_signal
