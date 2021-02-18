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

import json
import os
import sys
from copy import deepcopy

from cortex_internal.lib import util
from cortex_internal.lib.api import get_spec, TaskAPI
from cortex_internal.lib.exceptions import UserRuntimeException
from cortex_internal.lib.telemetry import capture_exception, get_default_tags, init_sentry
from cortex_internal.lib.log import configure_logger

init_sentry(tags=get_default_tags())
logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


def start():
    cache_dir = os.environ["CORTEX_CACHE_DIR"]
    provider = os.environ["CORTEX_PROVIDER"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]
    region = os.getenv("AWS_REGION")

    api_spec_path = os.environ["CORTEX_API_SPEC"]
    task_spec_path = os.environ["CORTEX_TASK_SPEC"]

    _, api_spec = get_spec(provider, api_spec_path, cache_dir, region)
    _, task_spec = get_spec(provider, task_spec_path, cache_dir, region, spec_name="task-spec.json")

    logger.info("loading the task definition from {}".format(api_spec["definition"]["path"]))
    task_api = TaskAPI(provider, api_spec)

    logger.info("executing the task definition from {}".format(api_spec["definition"]["path"]))
    callable_fn = task_api.get_callable(project_dir)

    config = deepcopy(api_spec["definition"]["config"])
    if task_spec is not None and task_spec.get("config") is not None:
        util.merge_dicts_in_place_overwrite(config, task_spec["config"])

    try:
        callable_fn(config)
    except Exception as err:
        logger.error(f"failed to run task {task_spec['job_id']}", exc_info=True)
        sys.exit(1)


if __name__ == "__main__":
    start()
