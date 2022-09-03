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

import json
import math
import os
import re
import threading as td
import time
from http import HTTPStatus
from pathlib import Path
from typing import Callable, Dict, Any, List, Union

import boto3
import cortex as cx
import requests
import yaml

from e2e.expectations import (
    parse_expectations,
    assert_response_expectations,
    assert_json_expectations,
)
from e2e.generator import load_generator
from e2e.utils import (
    apis_ready,
    api_updated,
    wait_on_event,
    wait_on_futures,
    endpoint_ready,
    post_request,
    get_request,
    job_done,
    jobs_done,
    request_batch_prediction,
    request_task,
    retrieve_async_result,
    make_requests_concurrently,
    check_futures_healthy,
    retrieve_results_concurrently,
    stream_api_logs,
    stream_job_logs,
    wait_for,
)

TEST_APIS_DIR = Path(__file__).parent.parent.parent / "apis"


def delete_apis(client: cx.Client, api_names: List[str]):
    for name in api_names:
        client.delete(name)


def test_realtime_api(
    printer: Callable,
    client: cx.Client,
    api: str,
    timeout: int = None,
    api_config_name: str = "cortex_cpu.yaml",
    node_groups: List[str] = [],
    extra_path: str = "",
    method: str = "POST",
):
    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)
    assert len(api_specs) == 1

    if len(node_groups) > 0:
        api_specs[0]["node_groups"] = node_groups

    expectations = None
    expectations_file = api_dir / "expectations.yaml"
    if expectations_file.exists():
        expectations = parse_expectations(str(expectations_file))

    api_name = api_specs[0]["name"]
    for api_spec in api_specs:
        client.deploy(api_spec=api_spec)

    try:
        assert apis_ready(
            client=client, api_names=[api_name], timeout=timeout
        ), f"apis {api_name} not ready"

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)
        if method == "POST":
            response = post_request(client, api_name, payload, extra_path)
        else:
            response = get_request(client, api_name, payload, extra_path)

        assert (
            response.status_code == HTTPStatus.OK
        ), f"status code: got {response.status_code}, expected {HTTPStatus.OK}"

        if expectations and "response" in expectations:
            assert_response_expectations(response, expectations["response"])
    except:
        # best effort
        try:
            api_info = client.get_api(api_name)
            printer(json.dumps(api_info, indent=2))
            td.Thread(target=lambda: stream_api_logs(client, api_name), daemon=True).start()
            time.sleep(5)
        finally:
            raise
    finally:
        delete_apis(client, [api_name])


def test_batch_api(
    printer: Callable,
    client: cx.Client,
    api: str,
    test_s3_path: str,
    deploy_timeout: int = None,
    job_timeout: int = None,
    retry_attempts: int = 0,
    api_config_name: str = "cortex_cpu.yaml",
    node_groups: List[str] = [],
    local_operator: bool = False,
):
    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)
    assert len(api_specs) == 1

    if len(node_groups) > 0:
        api_specs[0]["node_groups"] = node_groups

    api_name = api_specs[0]["name"]
    client.deploy(api_spec=api_specs[0])

    try:
        endpoint_override = f"http://localhost:8888/batch/{api_name}" if local_operator else None
        assert endpoint_ready(
            client=client,
            api_name=api_name,
            timeout=deploy_timeout,
            endpoint_override=endpoint_override,
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
                local_operator=local_operator,
            )
            if response.status_code == HTTPStatus.OK:
                break

            time.sleep(1)

        assert (
            response.status_code == HTTPStatus.OK
        ), f"status code: got {response.status_code}, expected {HTTPStatus.OK} ({response.text})"

        job_spec = response.json()

        # monitor job progress
        job_id = job_spec["job_id"]
        endpoint_override = (
            f"http://localhost:8888/batch/{api_name}?jobID={job_id}" if local_operator else None
        )
        assert job_done(
            client=client,
            api_name=api_name,
            job_id=job_id,
            timeout=job_timeout,
            endpoint_override=endpoint_override,
        ), f"job did not succeed (api_name: {api_name}, job_id: {job_spec['job_id']})"

    except:
        # best effort
        try:
            api_info = client.get_api(api_name)
            printer(json.dumps(api_info, indent=2))

            job_status = client.get_job(api_name, job_spec["job_id"])
            printer(json.dumps(job_status, indent=2))

            td.Thread(
                target=lambda: stream_job_logs(client, api_name, job_spec["job_id"]),
                daemon=True,
            ).start()
            time.sleep(5)
        finally:
            raise
    finally:
        delete_apis(client, [api_name])


def test_async_api(
    printer: Callable,
    client: cx.Client,
    api: str,
    deploy_timeout: int = None,
    poll_retries: int = 5,
    poll_sleep_seconds: int = 1,
    api_config_name: str = "cortex_cpu.yaml",
    node_groups: List[str] = [],
):
    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)
    assert len(api_specs) == 1

    if len(node_groups) > 0:
        api_specs[0]["node_groups"] = node_groups

    expectations = None
    expectations_file = api_dir / "expectations.yaml"
    if expectations_file.exists():
        expectations = parse_expectations(str(expectations_file))

    api_name = api_specs[0]["name"]
    client.deploy(api_spec=api_specs[0])

    try:
        assert apis_ready(
            client=client, api_names=[api_name], timeout=deploy_timeout
        ), f"apis {api_name} not ready"

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)

        response = post_request(client, api_name, payload)

        assert (
            response.status_code == HTTPStatus.OK
        ), f"workload submission status code: got {response.status_code}, expected {HTTPStatus.OK}"

        response_json = response.json()
        assert "id" in response_json

        request_id = response_json["id"]

        result_response = None
        for _ in range(poll_retries + 1):
            result_response = retrieve_async_result(
                client=client, api_name=api_name, request_id=request_id
            )

            if result_response.status_code != HTTPStatus.OK:
                time.sleep(poll_sleep_seconds)
                continue

            result_response_json = result_response.json()
            assert (
                "id" in result_response_json
            ), f"id key was not present in result response (response: {result_response_json})"
            assert (
                "status" in result_response_json
            ), f"status key was not present in result response (response: {result_response_json})"

            if result_response_json["status"] != "completed":
                time.sleep(poll_sleep_seconds)
                continue
            break

        assert (
            result_response.status_code == HTTPStatus.OK
        ), f"result retrieval status code: got {result_response.status_code}, expected {HTTPStatus.OK}"

        # validate keys are in the result json response
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

    except:
        # best effort
        try:
            api_info = client.get_api(api_name)
            printer(json.dumps(api_info, indent=2))
            printer(json.dumps(result_response_json, indent=2))
            td.Thread(
                target=lambda: stream_api_logs(client, api_name),
                daemon=True,
            ).start()
            time.sleep(5)
        except:
            pass
        raise

    finally:
        delete_apis(client, [api_name])


def test_task_api(
    printer: Callable,
    client: cx.Client,
    api: str,
    deploy_timeout: int = None,
    job_timeout: int = None,
    retry_attempts: int = 0,
    api_config_name: str = "cortex_cpu.yaml",
    node_groups: List[str] = [],
    local_operator: bool = False,
):
    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)

    assert len(api_specs) == 1

    if len(node_groups) > 0:
        api_specs[0]["node_groups"] = node_groups

    api_name = api_specs[0]["name"]
    client.deploy(api_spec=api_specs[0])

    try:
        endpoint_override = f"http://localhost:8888/tasks/{api_name}" if local_operator else None
        assert endpoint_ready(
            client=client,
            api_name=api_name,
            timeout=deploy_timeout,
            endpoint_override=endpoint_override,
        ), f"api {api_name} not ready"

        response = None
        for _ in range(retry_attempts + 1):
            response = request_task(client, api_name, local_operator=local_operator)
            if response.status_code == HTTPStatus.OK:
                break

            time.sleep(1)

        job_spec = response.json()

        job_id = job_spec["job_id"]
        endpoint_override = (
            f"http://localhost:8888/tasks/{api_name}?jobID={job_id}" if local_operator else None
        )
        assert job_done(
            client=client,
            api_name=api_name,
            job_id=job_spec["job_id"],
            timeout=job_timeout,
            endpoint_override=endpoint_override,
        ), f"task job did not succeed (api_name: {api_name}, job_id: {job_spec['job_id']})"

    except:
        # best effort
        try:
            api_info = client.get_api(api_name)
            printer(json.dumps(api_info, indent=2))

            job_status = client.get_job(api_name, job_spec["job_id"])
            printer(json.dumps(job_status, indent=2))
            td.Thread(
                target=lambda: stream_job_logs(client, api_name, job_spec["job_id"]), daemon=True
            ).start()
            time.sleep(5)
        except:
            pass
        raise

    finally:
        delete_apis(client, [api_name])


def test_autoscaling(
    printer: Callable,
    client: cx.Client,
    apis: Dict[str, Any],
    autoscaling_config: Dict[str, Union[int, float]],
    deploy_timeout: int = None,
    api_config_name: str = "cortex_cpu.yaml",
    node_groups: List[str] = [],
):
    max_replicas = autoscaling_config["max_replicas"]
    query_params = apis["query_params"]

    # increase the concurrency by 1 to ensure we get max_replicas replicas
    concurrency = max_replicas + 1

    all_apis = [apis["primary"]] + apis["dummy"]
    all_api_names = []
    for api in all_apis:
        api_dir = TEST_APIS_DIR / api
        with open(str(api_dir / api_config_name)) as f:
            api_specs = yaml.safe_load(f)
        assert len(api_specs) == 1

        if len(node_groups) > 0:
            api_specs[0]["node_groups"] = node_groups
        api_specs[0]["autoscaling"] = {
            "max_replicas": max_replicas,
            "downscale_stabilization_period": "1m",
        }

        all_api_names.append(api_specs[0]["name"])
        client.deploy(api_spec=api_specs[0])

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

        threads_futures = make_requests_concurrently(
            client, primary_api_name, concurrency, request_stopper, query_params=query_params
        )

        test_start_time = time.time()

        # upscale/downscale the api
        printer(f"scaling up to {max_replicas} replicas")
        while True:
            assert api_updated(
                client, primary_api_name, timeout=deploy_timeout
            ), "api didn't scale up to the desired number of replicas in time"
            current_replicas = client.get_api(primary_api_name)["status"]["requested"]

            # stop the requests from being made
            if current_replicas == max_replicas and not request_stopper.is_set():
                printer(f"scaling back down to 1 replica")
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

    except:
        # best effort
        try:
            api_info = client.get_api(primary_api_name)
            printer(json.dumps(api_info, indent=2))
        finally:
            raise
    finally:
        request_stopper.set()
        delete_apis(client, all_api_names)


def test_load_realtime(
    printer: Callable,
    client: cx.Client,
    api: str,
    load_config: Dict[str, Union[int, float]],
    deploy_timeout: int = None,
    api_config_name: str = "cortex_cpu.yaml",
    node_groups: List[str] = [],
):

    total_requests = load_config["total_requests"]
    desired_replicas = load_config["desired_replicas"]
    concurrency = load_config["concurrency"]
    status_code_timeout = load_config["status_code_timeout"]

    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)
    assert len(api_specs) == 1

    api_specs[0]["autoscaling"] = {
        "min_replicas": desired_replicas,
        "max_replicas": desired_replicas,
    }
    if len(node_groups) > 0:
        api_specs[0]["node_groups"] = node_groups

    api_name = api_specs[0]["name"]
    client.deploy(api_spec=api_specs[0])

    # controls the flow of requests
    request_stopper = td.Event()
    failed = False
    try:
        printer(f"getting {desired_replicas} replicas ready")
        assert apis_ready(
            client=client, api_names=[api_name], timeout=deploy_timeout
        ), f"api {api_name} not ready"

        # give the APIs some time to prevent getting high latency spikes in the beginning
        time.sleep(5)

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)

        printer("start making requests concurrently")
        threads_futures = make_requests_concurrently(
            client,
            api_name,
            concurrency,
            request_stopper,
            max_total_requests=total_requests,
            payload=payload,
        )

        while not request_stopper.is_set():
            # check if the requesting threads are still healthy
            # if not, they'll raise an exception
            check_futures_healthy(threads_futures)

            # don't stress the CPU too hard
            time.sleep(1)
    except:
        # best effort
        failed = True
        try:
            api_info = client.get_api(api_name)
            printer(json.dumps(api_info, indent=2))
        finally:
            raise

    finally:
        request_stopper.set()
        delete_apis(client, [api_name])
        if failed:
            time.sleep(30)


def test_load_async(
    printer: Callable,
    client: cx.Client,
    api: str,
    load_config: Dict[str, Union[int, float]],
    deploy_timeout: int = None,
    poll_sleep_seconds: int = 1,
    api_config_name: str = "cortex_cpu.yaml",
    node_groups: List[str] = [],
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
    if len(node_groups) > 0:
        api_specs[0]["node_groups"] = node_groups

    api_name = api_specs[0]["name"]
    client.deploy(api_spec=api_specs[0])

    request_stopper = td.Event()
    map_stopper = td.Event()
    responses: List[Dict[str, Any]] = []
    failed = False
    try:
        printer(f"getting {desired_replicas} replicas ready")
        assert apis_ready(
            client=client, api_names=[api_name], timeout=deploy_timeout
        ), f"api {api_name} not ready"

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)

        printer("start making prediction requests concurrently")
        threads_futures = make_requests_concurrently(
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

        printer("finished making prediction requests")
        assert (
            len(responses) == total_requests
        ), f"the submitted number of requests doesn't match the returned number of responses"

        job_ids = []
        for response in responses:
            response_json = response.json()
            assert "id" in response_json
            job_ids.append(response_json["id"])

        # assert the results
        printer("start retrieving the async results concurrently")
        results = []
        retrieve_results_concurrently(
            client,
            api_name,
            concurrency,
            map_stopper,
            job_ids,
            results,
            poll_sleep_seconds,
            workload_timeout,
        )
        for request_id, json_result in results:
            # validate keys are in the result json response
            assert (
                "id" in json_result
            ), f"id key was not present in result response (response: {json_result})"
            assert (
                "result" in json_result
            ), f"result key was not present in result response (response: {json_result})"
            assert (
                "timestamp" in json_result
            ), f"timestamp key was not present in result response (response: {json_result})"

            # validate result json response has valid values
            assert (
                json_result["id"] == request_id
            ), f"result 'id' and request 'id' mismatch ({json_result['id']} != {request_id})"
            assert (
                json_result["status"] == "completed"
            ), f"async workload did not complete (response: {json_result})"
            assert json_result["timestamp"] != "", "result 'timestamp' value was empty"
            assert json_result["result"] != "", "result 'result' value was empty"

    except:
        # best effort
        failed = True
        try:
            api_info = client.get_api(api_name)
            printer(json.dumps(api_info, indent=2))
        finally:
            raise

    finally:
        if "results" in vars() and len(results) < total_requests:
            printer(f"{len(results)}/{total_requests} have been successfully retrieved")
        map_stopper.set()
        delete_apis(client, [api_name])
        if failed:
            time.sleep(30)


def test_load_batch(
    printer: Callable,
    client: cx.Client,
    api: str,
    test_s3_path: str,
    load_config: Dict[str, Union[int, float]],
    deploy_timeout: int = None,
    retry_attempts: int = 0,
    api_config_name: str = "cortex_cpu.yaml",
    node_groups: List[str] = [],
):

    jobs = load_config["jobs"]
    workers_per_job = load_config["workers_per_job"]
    items_per_job = load_config["items_per_job"]
    batch_size = load_config["batch_size"]
    workload_timeout = load_config["workload_timeout"]

    bucket, key = re.match("s3://(.+?)/(.+)", test_s3_path).groups()
    s3 = boto3.client("s3")
    paginator = s3.get_paginator("list_objects_v2")

    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)
    assert len(api_specs) == 1

    if len(node_groups) > 0:
        api_specs[0]["node_groups"] = node_groups

    sample_generator_path = api_dir / "sample_generator.py"
    assert (
        sample_generator_path.exists()
    ), "sample_generator.py must be present for the batch load test"
    sample_generator = load_generator(sample_generator_path)

    api_name = api_specs[0]["name"]
    client.deploy(api_spec=api_specs[0])
    api_endpoint = client.get_api(api_name)["endpoint"]
    failed = False
    try:
        assert endpoint_ready(
            client=client, api_name=api_name, timeout=deploy_timeout
        ), f"api {api_name} not ready"

        # submit jobs
        printer(f"submitting {jobs} jobs")
        job_specs = []
        for _ in range(jobs):
            for _ in range(retry_attempts + 1):
                response = request_batch_prediction(
                    client,
                    api_name,
                    item_list=[sample_generator() for _ in range(items_per_job)],
                    batch_size=batch_size,
                    workers=workers_per_job,
                    config={"dest_s3_dir": test_s3_path},
                )
                if response.status_code == HTTPStatus.OK:
                    break
                time.sleep(1)
            # retries are only required once
            retry_attempts = 0

            assert (
                response.status_code == HTTPStatus.OK
            ), f"status code: got {response.status_code}, expected {HTTPStatus.OK} ({response.text})"
            job_specs.append(response.json())

        # wait the jobs to finish
        printer("waiting on the jobs")
        assert jobs_done(
            client, api_name, [job_spec["job_id"] for job_spec in job_specs], workload_timeout
        ), f"not all jobs succeed in {workload_timeout}s"

        # assert jobs
        printer("checking the jobs' responses")
        for job_spec in job_specs:
            job_id: str = job_spec["job_id"]
            job = requests.get(f"{api_endpoint}?jobID={job_id}").json()
            job_status = job["job_status"]
            job_metrics = job["metrics"]

            assert (
                job_status["batches_in_queue"] == 0
            ), f"there are still batches in queue ({job_status['batches_in_queue']}) for job ID {job_id}"
            assert job_metrics["succeeded"] == math.ceil(items_per_job / batch_size)

            num_objects = 0
            for page in paginator.paginate(Bucket=bucket, Prefix=os.path.join(key, job_id)):
                num_objects += len(page["Contents"])
            assert num_objects == 1

    except:
        # best effort
        failed = True
        try:
            api_info = client.get_api(api_name)

            # only get the last 10 job statuses
            if "batch_job_statuses" in api_info and len(api_info["batch_job_statuses"]) > 10:
                api_info["batch_job_statuses"] = api_info["batch_job_statuses"][-10:]

            printer(json.dumps(api_info, indent=2))
        finally:
            raise

    finally:
        delete_apis(client, [api_name])
        if failed:
            time.sleep(30)


def test_long_running_realtime(
    printer: Callable,
    client: cx.Client,
    api: str,
    long_running_config: Dict[str, Union[int, float]],
    deploy_timeout: int = None,
    api_config_name: str = "cortex_cpu.yaml",
    node_groups: List[str] = [],
):
    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)
    assert len(api_specs) == 1

    time_to_run = long_running_config["time_to_run"]

    if len(node_groups) > 0:
        api_specs[0]["node_groups"] = node_groups

    expectations = None
    expectations_file = api_dir / "expectations.yaml"
    if expectations_file.exists():
        expectations = parse_expectations(str(expectations_file))

    api_name = api_specs[0]["name"]
    for api_spec in api_specs:
        client.deploy(api_spec=api_spec)

    try:
        assert apis_ready(
            client=client, api_names=[api_name], timeout=deploy_timeout
        ), f"apis {api_name} not ready"

        with open(str(api_dir / "sample.json")) as f:
            payload = json.load(f)

        counter = 0
        start_time = time.time()
        while time.time() - start_time <= time_to_run:
            response = post_request(client, api_name, payload)

            assert (
                response.status_code == HTTPStatus.OK
            ), f"status code: got {response.status_code}, expected {HTTPStatus.OK}"

            if expectations and "response" in expectations:
                assert_response_expectations(response, expectations["response"])

            counter += 1

    except:
        # best effort
        try:
            api_info = client.get_api(api_name)
            printer(json.dumps(api_info, indent=2))
            td.Thread(target=lambda: stream_api_logs(client, api_name), daemon=True).start()
            time.sleep(5)
        finally:
            raise
    finally:
        delete_apis(client, [api_name])


def test_realtime_scale_to_zero(
    client: cx.Client,
    api: str,
    timeout: int = None,
    api_config_name: str = "cortex_cpu.yaml",
):
    api_dir = TEST_APIS_DIR / api
    with open(str(api_dir / api_config_name)) as f:
        api_specs = yaml.safe_load(f)

    api_name = api_specs[0]["name"]
    for api_spec in api_specs:
        client.deploy(api_spec=api_spec)

    try:
        assert apis_ready(
            client=client, api_names=[api_name], timeout=timeout, greater_or_equal_to=0
        ), f"apis {api_name} not ready"

        api_info = client.get_api(api_name)
        endpoint = api_info["endpoint"]

        response = requests.post(endpoint, json={}, timeout=30)

        # make first request, which should go to activator
        assert (
            response.status_code == HTTPStatus.OK
        ), f"status code: got {response.status_code}, expected {HTTPStatus.OK}"

        assert response.headers.get("x-cortex-origin") == "activator"

        def _make_request() -> bool:
            res = requests.post(endpoint, json={}, timeout=30)

            # make second request, which should go directly to the api
            assert (
                res.status_code == HTTPStatus.OK
            ), f"status code: got {res.status_code}, expected {HTTPStatus.OK}"

            return res.headers.get("x-cortex-origin") == "api"

        assert wait_for(_make_request, timeout=60)
    finally:
        delete_apis(client, [api_name])
