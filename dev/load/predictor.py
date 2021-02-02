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

import requests
import time
from cortex_internal.lib.log import logger as cortex_logger


class PythonPredictor:
    def __init__(self, config):
        num_success = 0
        num_fail = 0
        for i in range(config["num_requests"]):
            if i > 0:
                time.sleep(config["sleep"])
            try:
                # response = requests.get(config["endpoint"])
                response = requests.post(config["endpoint"], json=config["data"])
            except Exception as e:
                num_fail += 1
                cortex_logger.error(
                    e,
                    extra={
                        "error": True,
                        "request_number": i,
                    },
                )
                continue
            if response.status_code == 200:
                num_success += 1
                cortex_logger.info(
                    "successful request", extra={"request_success": True, "request_number": i}
                )
            else:
                num_fail += 1
                cortex_logger.error(
                    response.text,
                    extra={
                        "error": True,
                        "code": response.status_code,
                        "request_number": i,
                    },
                )

        cortex_logger.warn(
            "FINISHED",
            extra={"finished": True, "num_success": num_success, "num_fail": num_fail},
        )

    def predict(self, payload):
        return "ok"
