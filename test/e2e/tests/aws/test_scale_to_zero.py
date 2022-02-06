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

from typing import Dict

import pytest

import cortex as cx
import e2e.tests

TEST_APIS = [{"api": "realtime/hello-world", "config": "cortex_scale_to_zero.yaml"}]


@pytest.mark.parametrize("api", TEST_APIS, ids=[api["api"] for api in TEST_APIS])
@pytest.mark.usefixtures("client")
def test_scale_to_zero_realtime(config: Dict, client: cx.Client, api: Dict[str, str]):
    e2e.tests.test_realtime_scale_to_zero(
        client=client,
        api=api["api"],
        timeout=config["global"]["realtime_deploy_timeout"],
        api_config_name=api["config"],
    )
