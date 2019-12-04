# Copyright 2019 Cortex Labs, Inc.
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

import logging
import sys
import time

from cortex.lib import stringify
import datetime as dt


class MyFormatter(logging.Formatter):
    converter = dt.datetime.fromtimestamp

    def formatTime(self, record, datefmt):
        ct = self.converter(record.created)
        s = ct.strftime(datefmt)
        return s


current_logger = None


def register_logger(name):
    logger = logging.getLogger(name)
    handler = logging.StreamHandler(stream=sys.stdout)
    formatter = MyFormatter(
        fmt="%(asctime)s:cortex:%(levelname)s:%(message)s", datefmt="%Y-%m-%d %H:%M:%S.%f"
    )
    handler.setFormatter(formatter)

    logger.propagate = False
    logger.addHandler(handler)
    logger.setLevel(logging.DEBUG)
    return logger


def refresh_logger():
    global current_logger
    if current_logger is not None:
        current_logger.disabled = True
    current_logger = register_logger("{}-cortex".format(int(time.time() * 1000000)))


def cx_logger():
    return current_logger


def debug_obj(name, payload, debug):
    if not debug:
        return

    cx_logger().info("{}: {}".format(name, stringify.truncate(payload)))


refresh_logger()
