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

import json
import time
from http import HTTPStatus
from pathlib import Path
from typing import List

import cortex as cx
import yaml

from e2e.expectations import parse_expectations, assert_response_expectations
from e2e.utils import (
    apis_ready,
    endpoint_ready,
    request_prediction,
    job_done,
    request_batch_prediction,
)

TEST_APIS_DIR = Path(__file__).parent.parent.parent / "apis"


def delete_apis(client: cx.Client, api_names: List[str]):
    for name in api_names:
        client.delete_api(name)


def test_realtime_api(
    client: cx.Client, api: str, timeout: int = None, api_config_name: str = "cortex.yaml"
):
    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)

    expectations = None
    expectations_file = api_dir / "expectations.yaml"
    if expectations_file.exists():
        expectations = parse_expectations(str(expectations_file))

    api_name = api_specs[0]["name"]
    for api_spec in api_specs:
        client.create_api(api_spec=api_spec, project_dir=api_dir)

    try:
        assert apis_ready(
            client=client, api_names=[api_name], timeout=timeout
        ), f"apis {api_name} not ready"

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)

        response = request_prediction(client, api_name, payload)

        assert (
            response.status_code == HTTPStatus.OK
        ), f"status code: got {response.status_code}, expected {HTTPStatus.OK}"

        if expectations and "response" in expectations:
            assert_response_expectations(response, expectations["response"])
    finally:
        delete_apis(client, [api_name])


def test_batch_api(
    client: cx.Client,
    api: str,
    test_s3_path: str,
    deploy_timeout: int = None,
    job_timeout: int = None,
    retry_attempts: int = 0,
    api_config_name: str = "cortex.yaml",
):
    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)

    assert len(api_specs) == 1

    api_name = api_specs[0]["name"]
    client.create_api(api_spec=api_specs[0], project_dir=api_dir)

    try:
        assert endpoint_ready(
            client=client, api_name=api_name, timeout=deploy_timeout
        ), f"api {api_name} not ready"

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)

        for i in range(retry_attempts + 1):
            response = request_batch_prediction(
                client,
                api_name,
                item_list=payload,
                batch_size=2,
                config={"dest_s3_dir": test_s3_path},
            )
            if response.status_code == HTTPStatus.OK:
                break

            time.sleep(1)

        assert (
            response.status_code == HTTPStatus.OK
        ), f"status code: got {response.status_code}, expected {HTTPStatus.OK} ({response.text})"

        job_spec = response.json()

        # monitor job progress
        assert job_done(
            client=client,
            api_name=job_spec["api_name"],
            job_id=job_spec["job_id"],
            timeout=job_timeout,
        ), f"job did not succeed (api_name: {api_name}, job_id: {job_spec['job_id']})"

    finally:
        delete_apis(client, [api_name])
