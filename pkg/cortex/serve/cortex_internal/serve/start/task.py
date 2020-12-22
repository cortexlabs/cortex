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

import json
import os

from cortex.lib.api import get_spec
from cortex.lib.log import cx_logger as logger
from cortex.lib.storage import S3

local_cache = {
    "api_spec": None,
    "task_spec": None,
}


def get_task_spec(storage, cache_dir, job_spec_path):
    local_spec_path = os.path.join(cache_dir, "task_spec.json")
    _, key = S3.deconstruct_s3_path(job_spec_path)
    storage.download_file(key, local_spec_path)
    with open(local_spec_path) as f:
        return json.load(f)


def start():
    cache_dir = os.environ["CORTEX_CACHE_DIR"]
    provider = os.environ["CORTEX_PROVIDER"]
    api_spec_path = os.environ["CORTEX_API_SPEC"]
    task_spec_path = os.environ["CORTEX_TASK_SPEC"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]

    region = os.getenv("AWS_REGION")

    storage, api_spec = get_spec(provider, api_spec_path, cache_dir, region)
    task_spec = get_task_spec(storage, cache_dir, task_spec_path)

    logger().info("loading the task definition from {}".format(api_spec["definition"]["path"]))

    # TODO validate the task definition and execute the task

    local_cache["api_spec"] = api_spec
    local_cache["provider"] = provider
    local_cache["task_spec"] = task_spec
    print(task_spec)


if __name__ == "__main__":
    start()
