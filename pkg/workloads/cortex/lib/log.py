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

import logging
import sys
import time
import http

from cortex.lib import stringify
import datetime as dt


class CortexFormatter(logging.Formatter):
    converter = dt.datetime.fromtimestamp

    def formatTime(self, record, datefmt):
        ct = self.converter(record.created)
        s = ct.strftime(datefmt)
        return s


# https://github.com/encode/uvicorn/blob/master/uvicorn/logging.py
class CortexAccessFormatter(CortexFormatter):
    def get_path(self, scope):
        return scope.get("root_path", "") + scope["path"]

    def get_status_code(self, record):
        status_code = record.__dict__["status_code"]
        status_and_phrase = status_code

        try:
            status_phrase = http.HTTPStatus(status_code).phrase
            status_and_phrase = f"{status_code} {status_phrase}"
        except:
            pass

        return status_and_phrase

    def formatMessage(self, record):
        scope = record.__dict__["scope"]
        record.__dict__.update(
            {
                "status_code": self.get_status_code(record),
                "method": scope["method"],
                "path": self.get_path(scope),
            }
        )
        return super().formatMessage(record)


formatter_pid = CortexFormatter(
    fmt="%(asctime)s:cortex:pid-%(process)d:%(levelname)s:%(message)s",
    datefmt="%Y-%m-%d %H:%M:%S.%f",
)

formatter_no_pid = CortexFormatter(
    fmt="%(asctime)s:cortex:%(levelname)s:%(message)s", datefmt="%Y-%m-%d %H:%M:%S.%f"
)


current_logger = None


def register_logger(name, show_pid=True):
    logger = logging.getLogger(name)
    handler = logging.StreamHandler(stream=sys.stdout)
    if show_pid:
        formatter = formatter_pid
    else:
        formatter = formatter_no_pid

    handler.setFormatter(formatter)

    logger.propagate = False
    logger.addHandler(handler)
    logger.setLevel(logging.DEBUG)
    return logger


def refresh_logger(show_pid=True):
    global current_logger
    if current_logger is not None:
        current_logger.disabled = True
    current_logger = register_logger("{}-cortex".format(int(time.time() * 1000000)), show_pid)


def cx_logger():
    return current_logger


refresh_logger()
