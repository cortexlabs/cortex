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

TEST_APIS = ["realtime/text-generator"]


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS)
def test_long_running_realtime(printer: Callable, config: Dict, client: cx.Client, api: str):
    skip_load_test = config["global"].get("skip_long_running", False)
    if skip_load_test:
        pytest.skip("--skip-long-running flag detected, skipping long-running test")

    e2e.tests.test_long_running_realtime(
        printer,
        client,
        api,
        long_running_config=config["global"]["long_running_test_config"],
        deploy_timeout=config["global"]["realtime_deploy_timeout"],
        node_groups=config["aws"]["x86_nodegroups"],
    )
