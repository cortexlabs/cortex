# Copyright 2022 Cortex Labs, Inc.
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

from typing import Callable, Dict

import cortex as cx
import pytest

import e2e.tests

TEST_APIS = ["task/iris-classifier-trainer"]


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS)
def test_task_api(printer: Callable, config: Dict, client: cx.Client, api: str):
    e2e.tests.test_task_api(
        printer,
        client,
        api,
        retry_attempts=5,
        deploy_timeout=config["global"]["task_deploy_timeout"],
        job_timeout=config["global"]["task_job_timeout"],
        node_groups=config["aws"]["x86_nodegroups"],
        local_operator=config["global"]["local_operator"],
    )
