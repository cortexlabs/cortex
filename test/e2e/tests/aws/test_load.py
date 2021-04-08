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

from typing import Dict

import cortex as cx
import pytest

import e2e.tests

TEST_APIS_REALTIME = ["tensorflow/iris-classifier"]
TEST_APIS_ASYNC = ["async/iris-classifier"]


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_REALTIME)
def test_load_realtime(config: Dict, client: cx.Client, api: str):
    skip_load_test = config["global"]["load_test_config"].get("skip_load", False)
    if skip_load_test:
        pytest.skip("--skip-load flag detected, skipping load tests")

    e2e.tests.test_load_realtime(
        client,
        api,
        load_config=config["global"]["load_test_config"]["realtime"],
        deploy_timeout=config["global"]["realtime_deploy_timeout"],
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_ASYNC)
def test_load_async(config: Dict, client: cx.Client, api: str):
    skip_load_test = config["global"]["load_test_config"].get("skip_load", False)
    if skip_load_test:
        pytest.skip("--skip-load flag detected, skipping load tests")

    e2e.tests.test_load_async(
        client,
        api,
        load_config=config["global"]["load_test_config"]["async"],
        deploy_timeout=config["global"]["async_deploy_timeout"],
    )
