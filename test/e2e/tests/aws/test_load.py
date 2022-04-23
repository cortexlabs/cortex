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

TEST_APIS_REALTIME = ["realtime/prime-generator"]
TEST_APIS_ASYNC = ["async/text-generator"]
TEST_APIS_BATCH = ["batch/sum"]


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_REALTIME)
def test_load_realtime(printer: Callable, config: Dict, client: cx.Client, api: str):
    skip_load_test = config["global"].get("skip_load", False)
    if skip_load_test:
        pytest.skip("--skip-load flag detected, skipping load tests")

    e2e.tests.test_load_realtime(
        printer,
        client,
        api,
        load_config=config["global"]["load_test_config"]["realtime"],
        deploy_timeout=config["global"]["realtime_deploy_timeout"],
        node_groups=config["aws"]["x86_nodegroups"],
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_ASYNC)
def test_load_async(printer: Callable, config: Dict, client: cx.Client, api: str):
    skip_load_test = config["global"].get("skip_load", False)
    if skip_load_test:
        pytest.skip("--skip-load flag detected, skipping load tests")

    e2e.tests.test_load_async(
        printer,
        client,
        api,
        load_config=config["global"]["load_test_config"]["async"],
        deploy_timeout=config["global"]["async_deploy_timeout"],
        node_groups=config["aws"]["x86_nodegroups"],
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_BATCH)
def test_load_batch(printer: Callable, config: Dict, client: cx.Client, api: str):
    skip_load_test = config["global"].get("skip_load", False)
    if skip_load_test:
        pytest.skip("--skip-load flag detected, skipping load tests")

    s3_path = config["aws"].get("s3_path")
    if not s3_path:
        pytest.skip(
            "--s3-path option is required to run batch tests (alternatively set the "
            "CORTEX_TEST_BATCH_S3_PATH env var) )"
        )

    e2e.tests.test_load_batch(
        printer,
        client,
        api,
        test_s3_path=s3_path,
        load_config=config["global"]["load_test_config"]["batch"],
        deploy_timeout=config["global"]["batch_deploy_timeout"],
        node_groups=config["aws"]["x86_nodegroups"],
    )
