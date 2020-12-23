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

TIMEOUT_DEPLOY = 10  # seconds
JOB_TIMEOUT = 30  # seconds
TEST_APIS = ["batch/image-classifier"]


@pytest.fixture
def s3_bucket(request):
    s3_bucket = request.config.getoption("--s3-bucket")
    if not s3_bucket:
        pytest.skip("--s3-bucket option is required to run batch tests")

    return s3_bucket


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS)
def test_batch_api(client: cx.Client, api: str, s3_bucket: str):
    e2e.tests.test_batch_api(client, api, deploy_timeout=TIMEOUT_DEPLOY, job_timeout=JOB_TIMEOUT)

