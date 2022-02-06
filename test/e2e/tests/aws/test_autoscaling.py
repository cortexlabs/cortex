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

from typing import Any, Callable, Dict

import cortex as cx
import pytest

import e2e.tests

TEST_APIS = [
    {
        "primary": "realtime/sleep",
        "dummy": ["realtime/prime-generator"],
        "query_params": {
            "sleep": "1.0",
        },
    }
]


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("apis", TEST_APIS, ids=[api["primary"] for api in TEST_APIS])
def test_autoscaling(printer: Callable, config: Dict, client: cx.Client, apis: Dict[str, Any]):
    skip_autoscaling_test = config["global"].get("skip_autoscaling", False)
    if skip_autoscaling_test:
        pytest.skip("--skip-autoscaling flag detected, skipping autoscaling tests")

    e2e.tests.test_autoscaling(
        printer,
        client,
        apis,
        autoscaling_config=config["global"]["autoscaling_test_config"],
        deploy_timeout=config["global"]["realtime_deploy_timeout"],
        node_groups=config["aws"]["x86_nodegroups"],
    )
