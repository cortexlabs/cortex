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
import os

import cortex as cx
import pytest

import e2e.tests

DEPLOY_TIMEOUT = int(os.environ.get("CORTEX_TEST_BATCH_DEPLOY_TIMEOUT", 30))  # seconds
JOB_TIMEOUT = int(os.environ.get("CORTEX_TEST_BATCH_JOB_TIMEOUT", 120))  # seconds
TEST_APIS = ["batch/image-classifier", "batch/onnx", "batch/tensorflow"]


@pytest.fixture
def s3_bucket(request):
    s3_bucket = os.environ.get("CORTEX_TEST_BATCH_S3_BUCKET_DIR")
    s3_bucket = request.config.getoption("--s3-bucket") if s3_bucket is None else s3_bucket
    if not s3_bucket:
        pytest.skip(
            "--s3-bucket option is required to run batch tests (alternatively set the "
            "CORTEX_TEST_BATCH_S3_BUCKET_DIR env var) )"
        )

    return s3_bucket


@pytest.mark.usefixtures("client")
@pytest.mark.parametrize("api", TEST_APIS)
def test_batch_api(client: cx.Client, api: str, s3_bucket: str):
    e2e.tests.test_batch_api(
        client,
        api,
        test_bucket=s3_bucket,
        deploy_timeout=DEPLOY_TIMEOUT,
        job_timeout=JOB_TIMEOUT,
        retry_attempts=5,
    )
