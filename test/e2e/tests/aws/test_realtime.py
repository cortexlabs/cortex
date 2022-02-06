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

TEST_APIS = [
    {
        "name": "realtime/image-classifier-resnet50",
        "extra_path": "v1/models/resnet50:predict",
    },
    {
        "name": "realtime/prime-generator",
        "extra_path": "",
    },
    {
        "name": "realtime/text-generator",
        "extra_path": "",
    },
]
TEST_APIS_ARM = [
    {
        "name": "realtime/hello-world",
        "extra_path": "",
    },
]
TEST_APIS_GPU = [
    {
        "name": "realtime/image-classifier-resnet50",
        "extra_path": "v1/models/resnet50:predict",
    },
    {
        "name": "realtime/text-generator",
        "extra_path": "",
    },
]


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS, ids=[api["name"] for api in TEST_APIS])
def test_realtime_api(printer: Callable, config: Dict, client: cx.Client, api: Dict[str, str]):
    e2e.tests.test_realtime_api(
        printer=printer,
        client=client,
        api=api["name"],
        timeout=config["global"]["realtime_deploy_timeout"],
        node_groups=config["aws"]["x86_nodegroups"],
        extra_path=api["extra_path"],
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_ARM, ids=[api["name"] for api in TEST_APIS_ARM])
def test_realtime_api_arm(printer: Callable, config: Dict, client: cx.Client, api: Dict[str, str]):
    e2e.tests.test_realtime_api(
        printer=printer,
        client=client,
        api=api["name"],
        timeout=config["global"]["realtime_deploy_timeout"],
        api_config_name="cortex_cpu_arm64.yaml",
        node_groups=config["aws"]["arm_nodegroups"],
        extra_path=api["extra_path"],
        method="GET",
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_GPU, ids=[api["name"] for api in TEST_APIS_GPU])
def test_realtime_api_gpu(printer: Callable, config: Dict, client: cx.Client, api: Dict[str, str]):
    skip_gpus = config["global"].get("skip_gpus", False)
    if skip_gpus:
        pytest.skip("--skip-gpus flag detected, skipping GPU tests")

    e2e.tests.test_realtime_api(
        printer=printer,
        client=client,
        api=api["name"],
        timeout=config["global"]["realtime_deploy_timeout"],
        api_config_name="cortex_gpu.yaml",
        node_groups=config["aws"]["x86_nodegroups"],
        extra_path=api["extra_path"],
    )
