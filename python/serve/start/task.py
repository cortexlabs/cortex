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
from cortex_internal.lib.api import TaskAPI
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.telemetry import get_default_tags, init_sentry

init_sentry(tags=get_default_tags())
logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


def start():
    project_dir = os.environ["CORTEX_PROJECT_DIR"]
    api_spec_path = os.environ["CORTEX_API_SPEC"]
    task_spec_path = os.environ["CORTEX_TASK_SPEC"]

    with open(api_spec_path) as json_file:
        api_spec = json.load(json_file)

    with open(task_spec_path) as json_file:
        task_spec = json.load(json_file)

    logger.info("loading the task definition from {}".format(api_spec["definition"]["path"]))
    task_api = TaskAPI(api_spec)

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
