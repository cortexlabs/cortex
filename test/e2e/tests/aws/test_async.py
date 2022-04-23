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

TEST_APIS = ["async/text-generator"]
TEST_APIS_GPU = ["async/text-generator"]


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS)
def test_async_api(printer: Callable, config: Dict, client: cx.Client, api: str):
    e2e.tests.test_async_api(
        printer=printer,
        client=client,
        api=api,
        deploy_timeout=config["global"]["async_deploy_timeout"],
        poll_retries=config["global"]["async_workload_timeout"],
        node_groups=config["aws"]["x86_nodegroups"],
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_GPU)
def test_async_api_gpu(printer: Callable, config: Dict, client: cx.Client, api: str):
    skip_gpus = config["global"].get("skip_gpus", False)
    if skip_gpus:
        pytest.skip("--skip-gpus flag detected, skipping GPU tests")

    e2e.tests.test_async_api(
        printer=printer,
        client=client,
        api=api,
        deploy_timeout=config["global"]["async_deploy_timeout"],
        poll_retries=config["global"]["async_workload_timeout"],
        api_config_name="cortex_gpu.yaml",
        node_groups=config["aws"]["x86_nodegroups"],
    )
