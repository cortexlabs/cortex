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


import os
import base64
import time

import datadog

from cortex.lib.storage import S3
from cortex import consts
from cortex.lib import util
from cortex.lib.exceptions import UserException, CortexException
from cortex.lib.log import cx_logger


API_SUMMARY_MESSAGE = (
    "make a prediction by sending a post request to this endpoint with a json payload"
)


def assert_api_version():
    if os.environ["CORTEX_VERSION"] != consts.CORTEX_VERSION:
        raise ValueError(
            "api version mismatch (context: {}, image: {})".format(
                os.environ["CORTEX_VERSION"], consts.CORTEX_VERSION
            )
        )


def get_spec(cache_dir, s3_path):
    local_spec_path = os.path.join(cache_dir, "api_spec.msgpack")
    bucket, key = S3.deconstruct_s3_path(s3_path)
    S3(bucket, client_config={}).download_file(key, local_spec_path)
    return util.read_msgpack(local_spec_path)


def get_statsd_client():
    host_ip = os.environ["HOST_IP"]
    datadog.initialize(statsd_host=host_ip, statsd_port="8125")
    return datadog.statsd


def extract_waitress_params(api):
    waitress_kwargs = {}
    if api.spec["predictor"].get("config") is not None:
        for key, value in api.spec["predictor"]["config"].items():
            if key.startswith("waitress_"):
                waitress_kwargs[key[len("waitress_") :]] = value

    if len(waitress_kwargs) > 0:
        cx_logger().info("waitress parameters: {}".format(waitress_kwargs))

    return waitress_kwargs
