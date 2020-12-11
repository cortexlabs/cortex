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

import cortex as cx
import pytest

import e2e.tests
from e2e.utils import client_from_config

TEST_APIS = ["pytorch/iris-classifier", "pytorch/image-classifier-resnet50"]
DEPLOY_TIMEOUT = 30  # seconds


@pytest.fixture
def client(request):
    env_name = request.config.getoption("--aws-env")
    if env_name:
        return cx.client(env_name)

    config_path = request.config.getoption("--aws-config")
    if config_path is None:
        raise ValueError(
            "missing arguments: --env-name <name> or --aws-config <aws_cluster_config>"
        )

    return client_from_config(config_path)


@pytest.mark.parametrize("api", TEST_APIS)
def test_realtime_api(client: cx.Client, api: str):
    e2e.tests.test_realtime_api(client=client, api=api, timeout=DEPLOY_TIMEOUT)
