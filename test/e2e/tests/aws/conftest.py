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

import cortex as cx
import pytest

import e2e
from e2e.utils import client_from_config


@pytest.fixture
def client(config):
    env_name = config["aws"]["env"]
    if env_name:
        return cx.client(env_name)

    config_path = config["aws"]["config"]
    if config_path is not None:
        return client_from_config(config_path)

    pytest.skip("--env or --config must be passed to run tests")


def pytest_configure(config):
    cluster_config = config.getoption("--config")
    if cluster_config:
        e2e.create_cluster(cluster_config)


def pytest_unconfigure(config):
    cluster_config = config.getoption("--config")
    if cluster_config:
        e2e.delete_cluster(cluster_config)
