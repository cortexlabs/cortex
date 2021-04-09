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
from typing import Callable, Dict

import cortex as cx
import pytest

import e2e.tests

TEST_APIS = ["pytorch/iris-classifier", "onnx/iris-classifier", "tensorflow/iris-classifier"]
TEST_APIS_GRPC = ["grpc/iris-classifier-sklearn", "grpc/prime-number-generator"]
TEST_APIS_GPU = ["pytorch/text-generator", "tensorflow/text-generator"]
TEST_APIS_INF = ["pytorch/image-classifier-resnet50"]


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS)
def test_realtime_api(printer: Callable, config: Dict, client: cx.Client, api: str):
    e2e.tests.test_realtime_api(
        printer=printer, client=client, api=api, timeout=config["global"]["realtime_deploy_timeout"]
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_GRPC)
def test_realtime_api_grpc(printer: Callable, config: Dict, client: cx.Client, api: str):
    e2e.tests.test_realtime_api(
        printer=printer,
        client=client,
        api=api,
        timeout=config["global"]["realtime_deploy_timeout"],
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_GPU)
def test_realtime_api_gpu(printer: Callable, config: Dict, client: cx.Client, api: str):
    skip_gpus = config["global"].get("skip_gpus", False)
    if skip_gpus:
        pytest.skip("--skip-gpus flag detected, skipping GPU tests")

    e2e.tests.test_realtime_api(
        printer=printer, client=client, api=api, timeout=config["global"]["realtime_deploy_timeout"]
    )


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS_INF)
def test_realtime_api_inf(printer: Callable, config: Dict, client: cx.Client, api: str):
    skip_infs = config["global"].get("skip_infs", False)
    if skip_infs:
        pytest.skip("--skip-infs flag detected, skipping Inferentia tests")

    e2e.tests.test_realtime_api(
        printer=printer,
        client=client,
        api=api,
        timeout=config["global"]["realtime_deploy_timeout"],
        api_config_name="cortex_inf.yaml",
    )
