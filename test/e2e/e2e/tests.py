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
import math
import requests
from http import HTTPStatus
from pathlib import Path
from typing import Dict, Any, List, Union

import cortex as cx
import yaml
import threading as td
from concurrent import futures

from e2e.expectations import (
    parse_expectations,
    assert_response_expectations,
    assert_json_expectations,
)
from e2e.utils import (
    apis_ready,
    api_updated,
    api_requests,
    wait_on_event,
    wait_on_futures,
    endpoint_ready,
    request_prediction,
    generate_grpc,
    job_done,
    request_batch_prediction,
    request_task,
    retrieve_async_result,
    request_concurrent_predictions,
    check_futures_healthy,
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

        if not expectations or "grpc" not in expectations:
            with open(str(api_dir / "sample.json")) as f:
                payload = json.load(f)
            response = request_prediction(client, api_name, payload)

            assert (
                response.status_code == HTTPStatus.OK
            ), f"status code: got {response.status_code}, expected {HTTPStatus.OK}"

            if expectations and "response" in expectations:
                assert_response_expectations(response, expectations["response"])

        if expectations and "grpc" in expectations:
            stub, input_sample, output_values, output_type, is_output_stream = generate_grpc(
                client, api_name, api_dir, expectations["grpc"]
            )
            if is_output_stream:
                for response, output_val in zip(stub.Predict(input_sample), output_values):
                    assert (
                        type(response) == output_type
                    ), f"didn't receive response of type {str(output_type)}, but received {str(type(response))}"
                    assert response == output_val, f"received {response} instead of {output_val}"
            else:
                response = stub.Predict(input_sample)
                assert (
                    type(stub.Predict(input_sample)) == output_type
                ), f"didn't receive response of type {str(output_type)}, but received {str(type(response))}"
                assert (
                    response == output_values[0]
                ), f"received {response} instead of {output_values[0]}"
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

        response = None
        for _ in range(retry_attempts + 1):
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


def test_async_api(
    client: cx.Client,
    api: str,
    deploy_timeout: int = None,
    poll_retries: int = 5,
    poll_sleep_seconds: int = 1,
    api_config_name: str = "cortex.yaml",
):
    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)

    expectations = None
    expectations_file = api_dir / "expectations.yaml"
    if expectations_file.exists():
        expectations = parse_expectations(str(expectations_file))

    assert len(api_specs) == 1

    api_name = api_specs[0]["name"]
    client.create_api(api_spec=api_specs[0], project_dir=api_dir)

    try:
        assert apis_ready(
            client=client, api_names=[api_name], timeout=deploy_timeout
        ), f"apis {api_name} not ready"

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)

        response = request_prediction(client, api_name, payload)

        assert (
            response.status_code == HTTPStatus.OK
        ), f"workload submission status code: got {response.status_code}, expected {HTTPStatus.OK}"

        response_json = response.json()
        assert "id" in response_json

        request_id = response_json["id"]

        result_response = None
        for i in range(poll_retries + 1):
            result_response = retrieve_async_result(
                client=client, api_name=api_name, request_id=request_id
            )

            if result_response.status_code == HTTPStatus.OK:
                break

            time.sleep(poll_sleep_seconds)

        assert (
            result_response.status_code == HTTPStatus.OK
        ), f"result retrieval status code: got {result_response.status_code}, expected {HTTPStatus.OK}"

        result_response_json = result_response.json()

        # validate keys are in the result json response
        assert (
            "id" in result_response_json
        ), f"id key was not present in result response (response: {result_response_json})"
        assert (
            "status" in result_response_json
        ), f"status key was not present in result response (response: {result_response_json})"
        assert (
            "result" in result_response_json
        ), f"result key was not present in result response (response: {result_response_json})"
        assert (
            "timestamp" in result_response_json
        ), f"timestamp key was not present in result response (response: {result_response_json})"

        # validate result json response has valid values
        assert (
            result_response_json["id"] == request_id
        ), f"result 'id' and request 'id' mismatch ({result_response_json['id']} != {request_id})"
        assert (
            result_response_json["status"] == "completed"
        ), f"async workload did not complete (response: {result_response_json})"
        assert result_response_json["timestamp"] != "", "result 'timestamp' value was empty"
        assert result_response_json["result"] != "", "result 'result' value was empty"

        # assert result expectations
        if expectations:
            assert_json_expectations(result_response_json["result"], expectations["response"])

    finally:
        delete_apis(client, [api_name])


def test_task_api(
    client: cx.Client,
    api: str,
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

        response = None
        for _ in range(retry_attempts + 1):
            response = request_task(
                client,
                api_name,
            )
            if response.status_code == HTTPStatus.OK:
                break

            time.sleep(1)

        job_spec = response.json()

        assert job_done(
            client=client,
            api_name=api_name,
            job_id=job_spec["job_id"],
            timeout=job_timeout,
        ), f"task job did not succeed (api_name: {api_name}, job_id: {job_spec['job_id']})"

    finally:
        delete_apis(client, [api_name])


def test_autoscaling(
    client: cx.Client,
    apis: Dict[str, Any],
    deploy_timeout: int = None,
    api_config_name: str = "cortex.yaml",
):
    # max number of concurrent requests
    max_replicas = 20
    # increase the concurrency by 1 to ensure we get max_replicas replicas
    concurrency = max_replicas + 1

    all_apis = [apis["primary"]] + apis["dummy"]
    all_api_names = []
    for api in all_apis:
        api_dir = TEST_APIS_DIR / api
        with open(str(api_dir / api_config_name)) as f:
            api_specs = yaml.safe_load(f)
        assert len(api_specs) == 1
        api_specs[0]["autoscaling"] = {
            "max_replicas": max_replicas,
            "downscale_stabilization_period": "1m",
        }
        all_api_names.append(api_specs[0]["name"])
        client.create_api(api_spec=api_specs[0], project_dir=api_dir)

    primary_api_name = all_api_names[0]
    autoscaling = client.get_api(primary_api_name)["spec"]["autoscaling"]

    # controls the flow of requests
    request_stopper = td.Event()

    # determine upscale/downscale replica requests
    current_replicas = 1  # starting number of replicas
    test_timeout = 0  # measured in seconds
    while current_replicas < max_replicas:
        upscale_ceil = math.ceil(current_replicas * autoscaling["max_upscale_factor"])
        if upscale_ceil > current_replicas + 1:
            current_replicas = upscale_ceil
        else:
            current_replicas += 1
        if current_replicas > max_replicas:
            current_replicas = max_replicas
        test_timeout += int(autoscaling["upscale_stabilization_period"] / (1000 ** 3))
    while current_replicas > 1:
        downscale_ceil = math.ceil(current_replicas * autoscaling["max_downscale_factor"])
        if downscale_ceil < current_replicas - 1:
            current_replicas = downscale_ceil
        else:
            current_replicas -= 1
        test_timeout += int(autoscaling["downscale_stabilization_period"] / (1000 ** 3))

    # add overhead to the test timeout to account for the process of downloading images or adding nodes to the cluster
    test_timeout *= 2

    try:
        assert apis_ready(
            client=client, api_names=all_api_names, timeout=deploy_timeout
        ), f"apis {all_api_names} not ready"

        threads_futures = request_concurrent_predictions(
            client, primary_api_name, concurrency, request_stopper, query_params={"sleep": "1.0"}
        )

        test_start_time = time.time()

        # upscale/downscale the api
        while True:
            assert api_updated(
                client, primary_api_name, timeout=deploy_timeout
            ), "api didn't scale up to the desired number of replicas in time"
            current_replicas = client.get_api(primary_api_name)["status"]["replica_counts"][
                "requested"
            ]

            # stop the requests from being made
            if current_replicas == max_replicas:
                request_stopper.set()

            # check if the requesting threads are still healthy
            # if not, they'll raise an exception
            check_futures_healthy(threads_futures)

            # check if the test is taking too much time
            assert (
                time.time() - test_start_time < test_timeout
            ), f"autoscaling test for api {primary_api_name} did not finish in {test_timeout}s; current number of replicas is {current_replicas}/{concurrency}"

            # stop the test if it has finished
            if current_replicas == 1 and request_stopper.is_set():
                break

            # add some delay to reduce the number of gets
            time.sleep(1)

    finally:
        request_stopper.set()
        delete_apis(client, all_api_names)


def test_load_realtime(
    client: cx.Client,
    api: str,
    load_config: Dict[str, Union[int, float]],
    deploy_timeout: int = None,
    api_config_name: str = "cortex.yaml",
):

    total_requests = load_config["total_requests"]
    desired_replicas = load_config["desired_replicas"]
    concurrency = load_config["concurrency"]
    min_rtt = load_config["min_rtt"]
    max_rtt = load_config["max_rtt"]
    avg_rtt = load_config["avg_rtt"]
    avg_rtt_tolerance = load_config["avg_rtt_tolerance"]
    status_code_timeout = load_config["status_code_timeout"]

    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)
    assert len(api_specs) == 1
    api_specs[0]["autoscaling"] = {
        "min_replicas": desired_replicas,
        "max_replicas": desired_replicas,
    }
    api_name = api_specs[0]["name"]
    client.create_api(api_spec=api_specs[0], project_dir=api_dir)

    # controls the flow of requests
    request_stopper = td.Event()
    latencies: List[float] = []
    try:
        assert apis_ready(
            client=client, api_names=[api_name], timeout=deploy_timeout
        ), f"api {api_name} not ready"

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)

        threads_futures = request_concurrent_predictions(
            client,
            api_name,
            concurrency,
            request_stopper,
            latencies=latencies,
            max_total_requests=total_requests,
            payload=payload,
        )

        # upscale/downscale the api
        while not request_stopper.is_set():
            current_min_rtt = min(latencies) if len(latencies) > 0 else min_rtt
            assert (
                current_min_rtt >= min_rtt
            ), f"min latency threshold hit; got {current_min_rtt}s, but the lowest accepted latency is {min_rtt}s"

            current_max_rtt = max(latencies) if len(latencies) > 0 else max_rtt
            assert (
                current_max_rtt <= max_rtt
            ), f"max latency threshold hit; got {current_max_rtt}s, but the highest accepted latency is {max_rtt}s"

            current_avg_rtt = sum(latencies) / len(latencies) if len(latencies) > 0 else avg_rtt
            assert (
                current_avg_rtt > avg_rtt - avg_rtt_tolerance
                and current_avg_rtt < avg_rtt + avg_rtt_tolerance
            ), f"avg latency ({current_avg_rtt}s) falls outside the expected range ({avg_rtt - avg_rtt_tolerance}s - {avg_rtt + avg_rtt_tolerance})"

            network_stats = client.get_api(api_name)["metrics"]["network_stats"]
            assert (
                network_stats["code_4xx"] == 0
            ), f"detected 4xx response codes ({network_stats['code_4xx']}) in cortex get"
            assert (
                network_stats["code_5xx"] == 0
            ), f"detected 5xx response codes ({network_stats['code_5xx']}) in cortex get"

            # check if the requesting threads are still healthy
            # if not, they'll raise an exception
            check_futures_healthy(threads_futures)

            # don't stress the CPU too hard
            time.sleep(1)

        assert api_requests(
            client, api_name, total_requests, timeout=status_code_timeout
        ), f"the number of 2xx response codes for api {api_name} doesn't match the expected number {total_requests}"

    finally:
        request_stopper.set()
        delete_apis(client, [api_name])


def test_load_async(
    client: cx.Client,
    api: str,
    load_config: Dict[str, Union[int, float]],
    deploy_timeout: int = None,
    poll_sleep_seconds: int = 1,
    api_config_name: str = "cortex.yaml",
):

    total_requests = load_config["total_requests"]
    desired_replicas = load_config["desired_replicas"]
    concurrency = load_config["concurrency"]
    submit_timeout = load_config["submit_timeout"]
    workload_timeout = load_config["workload_timeout"]

    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)

    assert len(api_specs) == 1
    api_specs[0]["autoscaling"] = {
        "min_replicas": desired_replicas,
        "max_replicas": desired_replicas,
    }
    api_name = api_specs[0]["name"]
    client.create_api(api_spec=api_specs[0], project_dir=api_dir)
    api_info = client.get_api(api_name)

    request_stopper = td.Event()
    map_stopper = td.Event()
    responses: List[Dict[str, Any]] = []
    try:
        assert apis_ready(
            client=client, api_names=[api_name], timeout=deploy_timeout
        ), f"api {api_name} not ready"

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)

        threads_futures = request_concurrent_predictions(
            client,
            api_name,
            concurrency,
            request_stopper,
            responses=responses,
            max_total_requests=total_requests,
            payload=payload,
        )
        assert wait_on_event(
            request_stopper, submit_timeout
        ), f"{total_requests} couldn't be submitted in {submit_timeout}s"
        check_futures_healthy(threads_futures)
        wait_on_futures(threads_futures)

        assert (
            len(responses) == total_requests
        ), f"the submitted number of requests doesn't match the returned number of responses"

        for response in responses:
            response_json = response.json()
            assert "id" in response_json

        exec = futures.ThreadPoolExecutor(concurrency)
        thread_local = td.local()

        def _get_session() -> requests.Session:
            if not hasattr(thread_local, "session"):
                thread_local.session = requests.Session()
            return thread_local.session

        def _async_retriever(response: requests.Response):
            session = _get_session()

            response_json = response.json()
            request_id = response_json["id"]

            while not map_stopper.is_set():
                result_response = session.get(f"{api_info['endpoint']}/{request_id}")

                if result_response.status_code != HTTPStatus.OK:
                    time.sleep(poll_sleep_seconds)
                    continue

                result_response_json = result_response.json()
                if (
                    "status" in result_response_json
                    and result_response_json["status"] == "completed"
                ):
                    break

                print(f"waiting on {request_id}")

            if map_stopper.is_set():
                return

            print(result_response_json)
            # validate keys are in the result json response
            assert (
                "id" in result_response_json
            ), f"id key was not present in result response (response: {result_response_json})"
            assert (
                "result" in result_response_json
            ), f"result key was not present in result response (response: {result_response_json})"
            assert (
                "timestamp" in result_response_json
            ), f"timestamp key was not present in result response (response: {result_response_json})"

            # validate result json response has valid values
            assert (
                result_response_json["id"] == request_id
            ), f"result 'id' and request 'id' mismatch ({result_response_json['id']} != {request_id})"
            assert (
                result_response_json["status"] == "completed"
            ), f"async workload did not complete (response: {result_response_json})"
            assert result_response_json["timestamp"] != "", "result 'timestamp' value was empty"
            assert result_response_json["result"] != "", "result 'result' value was empty"

        # will throw an exception if something failed in any thread
        for _ in exec.map(_async_retriever, responses, timeout=workload_timeout):
            pass

    finally:
        map_stopper.set()
        delete_apis(client, [api_name])
