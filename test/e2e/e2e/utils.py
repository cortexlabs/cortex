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

import concurrent
import sys
import time
import importlib
import pathlib
from http import HTTPStatus
from typing import Any, List, Optional, Tuple, Union, Dict, Callable

from concurrent import futures
import threading as td
import grpc
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


def api_updated(client: cx.Client, api_name: str, timeout: Optional[int] = None) -> bool:
    def _is_ready():
        status = client.get_api(api_name)["status"]
        return status["replica_counts"]["requested"] == status["replica_counts"]["updated"]["ready"]

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


def generate_grpc(
    client: cx.Client, api_name: str, api_dir: pathlib.Path, config: Dict[str, Any]
) -> Tuple[Any, Any, List, Any, bool]:
    api_info = client.get_api(api_name)

    test_proto_dir = pathlib.Path(config["proto_module_pb2"]).parent
    sys.path.append(str(api_dir / test_proto_dir))
    proto_module_pb2 = importlib.import_module(str(pathlib.Path(config["proto_module_pb2"]).stem))
    proto_module_pb2_grpc = importlib.import_module(
        str(pathlib.Path(config["proto_module_pb2_grpc"]).stem)
    )
    sys.path.pop()

    endpoint = api_info["endpoint"] + ":" + str(api_info["grpc_ports"]["insecure"])
    channel = grpc.insecure_channel(endpoint)
    stub = getattr(proto_module_pb2_grpc, config["stub_service_name"] + "Stub")(channel)

    input_sample = getattr(proto_module_pb2, config["input_spec"]["class_name"])()
    for k, v in config["input_spec"]["input"].items():
        setattr(input_sample, k, v)

    SampleClass = getattr(proto_module_pb2, config["output_spec"]["class_name"])
    output_values = []
    is_output_stream = config["output_spec"]["stream"]
    if is_output_stream:
        for entry in config["output_spec"]["output"]:
            output_val = SampleClass()
            for k, v in entry.items():
                setattr(output_val, k, v)
            output_values.append(output_val)
    else:
        output_val = SampleClass()
        for k, v in config["output_spec"]["output"].items():
            setattr(output_val, k, v)
        output_values.append(output_val)

    return stub, input_sample, output_values, SampleClass, is_output_stream


def request_prediction(
    client: cx.Client, api_name: str, payload: Union[List, Dict]
) -> requests.Response:
    api_info = client.get_api(api_name)
    response = requests.post(api_info["endpoint"], json=payload)

    return response


def retrieve_async_result(cliett: cx.Client, api_name: str, request_id: str) -> requests.Response:
    api_info = cliett.get_api(api_name)
    response = requests.get(f"{api_info['endpoint']}/{request_id}")

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


def request_task(
    client: cx.Client,
    api_name: str,
    config: Dict = None,
    timeout: int = None,
):
    api_info = client.get_api(api_name)
    endpoint = api_info["endpoint"]

    payload = {}
    if config is not None:
        payload["config"] = config
    if timeout is not None:
        payload["timeout"] = timeout

    response = requests.post(endpoint, json=payload)
    return response


def request_concurrent_predictions(
    client: cx.Client,
    api_name: str,
    concurrency: int,
    event_stopper: td.Event,
    payload: Optional[Union[List, Dict]] = None,
    query_params: Dict[str, str] = {},
) -> List[futures.Future]:
    thread_local = td.local()
    executor = futures.ThreadPoolExecutor(concurrency)
    api_info = client.get_api(api_name)

    def get_session() -> requests.Session:
        if not hasattr(thread_local, "session"):
            thread_local.session = requests.Session()
        return thread_local.session

    def runnable():
        session = get_session()
        while not event_stopper.is_set():
            response = session.post(api_info["endpoint"], json=payload, params=query_params)
            assert (
                response.status_code == HTTPStatus.OK
            ), f"status code: got {response.status_code}, expected {HTTPStatus.OK}"

    futures_list = []
    for _ in range(concurrency):
        future = executor.submit(runnable)
        futures_list.append(future)

    return futures_list


def client_from_config(config_path: str) -> cx.Client:
    with open(config_path) as f:
        config = yaml.safe_load(f)

    cluster_name = config["cluster_name"]

    return cx.client(f"{cluster_name}")
