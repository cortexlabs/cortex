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

import time
from http import HTTPStatus
from typing import List, Optional, Dict, Union, Callable

import cortex as cx
import requests
import yaml


def wait_for(fn: Callable[[], bool], timeout=None) -> bool:
    deadline = time.time() + timeout if timeout else None
    while True:
        if deadline is not None and time.time() > deadline:
            return False

        done = fn()
        if done:
            return True

        time.sleep(1)


def apis_ready(client: cx.Client, api_names: List[str], timeout: Optional[int] = None) -> bool:
    def _is_ready():
        return all(
            [client.get_api(name)["status"]["status_code"] == "status_live" for name in api_names]
        )

    return wait_for(_is_ready, timeout=timeout)


def endpoint_ready(client: cx.Client, api_name: str, timeout: int = None) -> bool:
    def _is_ready():
        endpoint = client.get_api(api_name)["endpoint"]
        response = requests.post(endpoint)
        return response.status_code == HTTPStatus.BAD_REQUEST

    return wait_for(_is_ready, timeout=timeout)


def job_done(client: cx.Client, api_name: str, job_id: str, timeout: int = None):
    def _is_ready():
        job_info = client.get_job(api_name, job_id)
        return job_info["job_status"]["status"] == "status_succeeded"

    return wait_for(_is_ready, timeout=timeout)


def request_prediction(
    client: cx.Client, api_name: str, payload: Union[List, Dict]
) -> requests.Response:
    api_info = client.get_api(api_name)
    response = requests.post(api_info["endpoint"], json=payload)

    return response


def request_batch_prediction(
    client: cx.Client,
    api_name: str,
    item_list: List,
    batch_size: int,
    workers: int = 1,
    config: Dict = None,
) -> requests.Response:
    api_info = client.get_api(api_name)
    endpoint = api_info["endpoint"]

    batch_payload = {
        "workers": workers,
        "item_list": {"items": item_list, "batch_size": batch_size},
        "config": config,
    }
    response = requests.post(endpoint, json=batch_payload)

    return response


def client_from_config(config_path: str) -> cx.Client:
    with open(config_path) as f:
        config = yaml.safe_load(f)

    cluster_name = config["cluster_name"]
    provider = config["provider"]

    return cx.client(f"{cluster_name}-{provider}")
