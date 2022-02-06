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

TEST_APIS = ["batch/image-classifier-alexnet"]
TEST_APIS_GPU = ["batch/image-classifier-alexnet"]


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS)
def test_batch_api(printer: Callable, config: Dict, client: cx.Client, api: str):
    s3_path = config["aws"].get("s3_path")
    if not s3_path:
        pytest.skip(
            "--s3-path option is required to run batch tests (alternatively set the "
            "CORTEX_TEST_BATCH_S3_PATH env var) )"
        )

    e2e.tests.test_batch_api(
        printer,
        client,
        api,
        test_s3_path=s3_path,
        deploy_timeout=config["global"]["batch_deploy_timeout"],
        job_timeout=config["global"]["batch_job_timeout"],
        retry_attempts=5,
        node_groups=config["aws"]["x86_nodegroups"],
        local_operator=config["global"]["local_operator"],
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_GPU)
def test_batch_api_gpu(printer: Callable, config: Dict, client: cx.Client, api: str):
    skip_gpus = config["global"].get("skip_gpus", False)
    if skip_gpus:
        pytest.skip("--skip-gpus flag detected, skipping GPU tests")

    s3_path = config["aws"].get("s3_path")
    if not s3_path:
        pytest.skip(
            "--s3-path option is required to run batch tests (alternatively set the "
            "CORTEX_TEST_BATCH_S3_PATH env var) )"
        )

    e2e.tests.test_batch_api(
        printer=printer,
        client=client,
        api=api,
        test_s3_path=s3_path,
        deploy_timeout=config["global"]["batch_deploy_timeout"],
        job_timeout=config["global"]["batch_job_timeout"],
        retry_attempts=5,
        api_config_name="cortex_gpu.yaml",
        node_groups=config["aws"]["x86_nodegroups"],
        local_operator=config["global"]["local_operator"],
    )
