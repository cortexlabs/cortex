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
import logging.config
import sys
import time
import http
import datetime as dt
import yaml

from cortex_internal.lib import stringify


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


logger = None


def configure_logger(name: str, config_file: str):
    global logger
    logger = retrieve_logger(name, config_file)
    return logger


def retrieve_logger(name: str, config_file: str):
    with open(config_file, "r") as f:
        config = yaml.safe_load(f.read())
        logging.config.dictConfig(config)
    return logging.getLogger(name)
