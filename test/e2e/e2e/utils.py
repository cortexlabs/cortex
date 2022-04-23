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

import os
import threading as td
import time
from concurrent import futures
from http import HTTPStatus
from typing import Any, List, Optional, Tuple, Union, Dict, Callable

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


def apis_ready(
    client: cx.Client,
    api_names: List[str],
    timeout: Optional[int] = None,
    greater_or_equal_to: int = 1,
) -> bool:
    def _check_liveness(status):
        return (
            status["requested"] >= greater_or_equal_to
            and status["requested"] == status["ready"] == status["up_to_date"]
        )

    def _is_ready():
        return all([_check_liveness(client.get_api(name)["status"]) for name in api_names])

    return wait_for(_is_ready, timeout=timeout)


def api_updated(client: cx.Client, api_name: str, timeout: Optional[int] = None) -> bool:
    def _is_ready():
        status = client.get_api(api_name)["status"]
        return status["requested"] == status["ready"]

    return wait_for(_is_ready, timeout=timeout)


def wait_on_event(event: td.Event, timeout: Optional[int] = None) -> bool:
    def _is_ready():
        return event.is_set()

    return wait_for(_is_ready, timeout=timeout)


def wait_on_futures(futures_list: List[futures.Future], timeout: Optional[int] = None):
    def _is_ready():
        return all([future.done() for future in futures_list])

    return wait_for(_is_ready, timeout=timeout)


def endpoint_ready(
    client: cx.Client, api_name: str, timeout: int = None, endpoint_override: str = None
) -> bool:
    def _is_ready():
        if endpoint_override:
            endpoint = endpoint_override
        else:
            endpoint = client.get_api(api_name)["endpoint"]

        response = requests.post(endpoint)
        return response.status_code == HTTPStatus.BAD_REQUEST

    return wait_for(_is_ready, timeout=timeout)


def job_done(
    client: cx.Client,
    api_name: str,
    job_id: str,
    timeout: int = None,
    endpoint_override: str = None,
) -> bool:
    def _is_ready():
        if endpoint_override:
            job_info = requests.get(endpoint_override)
            job_info = job_info.json()
            return job_info["job_status"]["status"] == "succeeded"

        job_info = client.get_job(api_name, job_id)
        return job_info["job_status"]["status"] == "succeeded"

    return wait_for(_is_ready, timeout=timeout)


def jobs_done(client: cx.Client, api_name: str, job_ids: List[str], timeout: int = None) -> bool:
    exec = futures.ThreadPoolExecutor(10)

    def _runnable(job_id):
        return job_done(client, api_name, job_id)

    try:
        for _ in exec.map(_runnable, job_ids, timeout=timeout):
            pass
    except:
        return False

    return True


def post_request(
    client: cx.Client,
    api_name: str,
    payload: Union[List, Dict],
    extra_path: Optional[str] = None,
) -> requests.Response:
    api_info = client.get_api(api_name)
    endpoint = api_info["endpoint"]
    if extra_path and extra_path != "":
        endpoint = os.path.join(endpoint, extra_path)
    response = requests.post(endpoint, json=payload)

    return response


def get_request(
    client: cx.Client,
    api_name: str,
    payload: Union[List, Dict],
    extra_path: Optional[str] = None,
) -> requests.Response:
    api_info = client.get_api(api_name)
    endpoint = api_info["endpoint"]
    if extra_path and extra_path != "":
        endpoint = os.path.join(endpoint, extra_path)
    response = requests.get(endpoint, json=payload)

    return response


def retrieve_async_result(client: cx.Client, api_name: str, request_id: str) -> requests.Response:
    api_info = client.get_api(api_name)
    response = requests.get(f"{api_info['endpoint']}/{request_id}")

    return response


def request_batch_prediction(
    client: cx.Client,
    api_name: str,
    item_list: List,
    batch_size: int,
    workers: int = 1,
    config: Dict = None,
    local_operator: bool = False,
) -> requests.Response:

    if local_operator:
        endpoint = f"http://localhost:8888/batch/{api_name}"
    else:
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
    local_operator: bool = False,
):
    if local_operator:
        endpoint = f"http://localhost:8888/tasks/{api_name}"
    else:
        api_info = client.get_api(api_name)
        endpoint = api_info["endpoint"]

    payload = {}
    if config is not None:
        payload["config"] = config
    if timeout is not None:
        payload["timeout"] = timeout

    response = requests.post(endpoint, json=payload)
    return response


_make_requests_concurrently_max_total_requests: int = 0


def make_requests_concurrently(
    client: cx.Client,
    api_name: str,
    concurrency: int,
    event_stopper: td.Event,
    latencies: Optional[List[float]] = None,
    responses: Optional[List[Dict[str, Any]]] = None,
    max_total_requests: Optional[int] = None,
    payload: Optional[Union[List, Dict]] = None,
    query_params: Dict[str, str] = {},
) -> List[futures.Future]:

    lock = td.RLock()
    thread_local = td.local()
    start_sync = td.Barrier(concurrency)
    end_sync = td.Barrier(concurrency)
    executor = futures.ThreadPoolExecutor(concurrency)
    api_info = client.get_api(api_name)
    endpoint = api_info["endpoint"]

    global _make_requests_concurrently_max_total_requests
    _make_requests_concurrently_max_total_requests = max_total_requests

    def get_session() -> requests.Session:
        if not hasattr(thread_local, "session"):
            thread_local.session = requests.Session()
        return thread_local.session

    def runnable():
        session = get_session()
        global _make_requests_concurrently_max_total_requests
        start_sync.wait()
        while not event_stopper.is_set():
            if _make_requests_concurrently_max_total_requests is not None:
                with lock:
                    if _make_requests_concurrently_max_total_requests == 0:
                        break
                    _make_requests_concurrently_max_total_requests -= 1
            start = time.time()
            response = session.post(endpoint, json=payload, params=query_params)
            assert (
                response.status_code == HTTPStatus.OK
            ), f"status code: got {response.status_code}, expected {HTTPStatus.OK}"
            if latencies is not None:
                latencies.append(time.time() - start)
            if responses is not None:
                responses.append(response)

        if _make_requests_concurrently_max_total_requests is not None:
            end_sync.wait()
            event_stopper.set()

    futures_list = []
    for _ in range(concurrency):
        future = executor.submit(runnable)
        futures_list.append(future)

    return futures_list


def retrieve_results_concurrently(
    client: cx.Client,
    api_name: str,
    concurrency: int,
    event_stopper: td.Event,
    job_ids: List[str],
    responses: List[Tuple[str, Dict[str, Any]]] = [],
    poll_sleep_seconds: int = 1,
    timeout: Optional[int] = None,
):
    api_info = client.get_api(api_name)
    task_kind = api_info["spec"]["kind"] == "TaskAPI"
    async_kind = api_info["spec"]["kind"] == "AsyncAPI"
    if not task_kind and not async_kind:
        raise ValueError("function can only be called for TaskAPI/AsyncAPI kinds")

    exec = futures.ThreadPoolExecutor(concurrency)
    thread_local = td.local()

    def _get_session() -> requests.Session:
        if not hasattr(thread_local, "session"):
            thread_local.session = requests.Session()
        return thread_local.session

    def _retriever(request_id: str):
        session = _get_session()

        while not event_stopper.is_set():
            if task_kind:
                result_response = session.get(f"{api_info['endpoint']}?jobID={request_id}")
            if async_kind:
                result_response = session.get(f"{api_info['endpoint']}/{request_id}")

            if result_response.status_code != HTTPStatus.OK:
                content = result_response.content.decode("utf-8")
                if "error" in content:
                    event_stopper.set()
                    raise RuntimeError(
                        f"received {result_response.status_code} status code with the following message: {content}"
                    )
                time.sleep(poll_sleep_seconds)
                continue

            result_response_json = result_response.json()
            if async_kind and "status" in result_response_json:
                if result_response_json["status"] == "completed":
                    break
                if result_response_json["status"] not in ["in_progress", "in_queue"]:
                    raise RuntimeError(
                        f"status for request ID {request_id} got set to {result_response_json['status']}"
                    )

            if (
                task_kind
                and "job_status" in result_response_json
                and "status" in result_response_json["job_status"]
            ):
                if result_response_json["job_status"]["status"] == "succeeded":
                    break
                if result_response_json["job_status"]["status"] not in [
                    "pending",
                    "enqueuing",
                    "running",
                ]:
                    raise RuntimeError(
                        f"status for job ID {request_id} got set to {result_response_json['job_status']['status']}"
                    )

        if event_stopper.is_set():
            return

        responses.append((request_id, result_response_json))

    # will throw an exception if something failed in any thread
    for _ in exec.map(_retriever, job_ids, timeout=timeout):
        pass


def check_futures_healthy(futures_list: List[futures.Future]):
    for future in futures_list:
        has_exception = None
        try:
            has_exception = future.exception(timeout=0.0)
        except futures.TimeoutError:
            pass
        if has_exception:
            future.result()


def client_from_config(config_path: str) -> cx.Client:
    with open(config_path) as f:
        config = yaml.safe_load(f)

    cluster_name = config["cluster_name"]

    return cx.client(f"{cluster_name}")


def stream_api_logs(client: cx.Client, api_name: str):
    cx.run_cli(["logs", api_name, "--random-pod", "-e", client.env_name])


def stream_job_logs(client: cx.Client, api_name: str, job_id: str):
    cx.run_cli(["logs", api_name, job_id, "--random-pod", "-e", client.env_name])
