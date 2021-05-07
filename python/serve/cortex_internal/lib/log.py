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

import http
import logging
import logging.config
import threading

import yaml
from pythonjsonlogger.jsonlogger import JsonFormatter


# https://github.com/encode/uvicorn/blob/master/uvicorn/logging.py
class CortexAccessFormatter(JsonFormatter):
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

    def format(self, record):
        if "scope" in record.__dict__:
            scope = record.__dict__["scope"]
            record.__dict__.update(
                {
                    "method": scope["method"],
                    "status_code": self.get_status_code(record),
                    "path": self.get_path(scope),
                }
            )
            record.__dict__.pop("scope", None)

        return super().format(record)


logger = None
_logger_initializer_mutex = threading.Lock()


def configure_logger(name: str, config_file: str):
    with _logger_initializer_mutex:
        global logger
        if logger is not None:
            return logger

        logger = retrieve_logger(name, config_file)
        return logger


def retrieve_logger(name: str, config_file: str):
    with open(config_file, "r") as f:
        config = yaml.safe_load(f.read())
        logging.config.dictConfig(config)
    return logging.getLogger(name)
