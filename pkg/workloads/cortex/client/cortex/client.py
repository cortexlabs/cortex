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

import json
import time
from typing import List, Dict, Optional, Tuple, Callable, Union
from cortex.binary import run_cli


def deploy(
    config_file: str,
    env: str,
    force: bool = False,
    wait: bool = False,
) -> list:
    """
    Deploy or update APIs.

    Args:
        config_file: Local path to a yaml file defining Cortex APIs.
        env: Environment to use.
        force: Override the in-progress api updates. Defaults to False.
        wait: Polls and streams logs until APIs are ready. Defaults to False.

    Returns:
        Deployment status, API specification and endpoint for each API.
    """

    args = [
        "deploy",
        config_file,
        "--env",
        env,
        "-o",
        "json",
    ]

    if force:
        args.append("--force")

    _, output = run_cli(
        args,
    )

    output_split = output.split("\n")

    deploy_results = []

    for line in output_split:
        if line.startswith("["):
            deploy_results = json.loads(line)

    if not wait:
        return deploy_results

    for deploy_result in deploy_results:
        api_name = deploy_result["api"]["spec"]["name"]
        kind = deploy_result["api"]["spec"]["kind"]
        print(kind)
        if kind != "RealtimeAPI":
            continue

        line = ""
        last_fetched = time.time()
        print("streaming")

        def stream_logs(c: str) -> bool:
            nonlocal last_fetched
            nonlocal line
            line += c
            if c != "\n":
                return True
            if "Uvicorn running on" in line:
                return False
            if time.time() - last_fetched > 2:
                api = get_api(api_name, env)
                last_fetched = time.time()
                if (
                    api["status"]["status_code"] != "status_live"
                    and api["status"]["status_code"] != "status_updating"
                ):
                    return False

            line = ""

            return True

        logs_api(api_name, env=env, stdout_iterator=stream_logs, hide_output=False)


def list_apis(env: Optional[str] = None) -> list:
    """
    List of information about APIs such as the API specification, endpoint, dashboard, status and metrics (if applicable).

    Args:
        env: List APIs in this environment. If unspecified, list APIs across all environments. Defaults to None.

    Returns:
        List of information about APIs.
    """
    args = ["get", "-o", "json"]

    if env is not None:
        args += ["--env", env]

    _, output = run_cli(args, hide_output=True)

    output_split = output.split("\n")

    for line in output_split:
        if line.startswith("[") or line.startswith("{"):
            return json.loads(line)

    return {}


def get_api(api_name: str, env: str) -> dict:
    """
    Get information about a specific API  such as the API specification, endpoint, dashboard, status and metrics (if applicable).

    Args:
        api_name: Name of the API
        env: Name of the environment hosting this API.

    Returns:
        Get of information about the specific API.
    """
    _, output = run_cli(["get", api_name, "--env", env, "-o", "json"], hide_output=True)

    output_split = output.split("\n")

    for line in output_split:
        if line.startswith("[") or line.startswith("{"):
            apis = json.loads(line)
            return apis[0]

    return {}


def get_job(api_name: str, job_id: str, env: str) -> dict:
    """
    Get information about a submitted job such as the job status, worker status and job progress.

    Args:
        api_name: Name of the Batch API.
        job_id: Job ID.
        env: Name of the environment hosting this Batch API.

    Returns:
        Information about the job.
    """
    _, output = run_cli(["get", api_name, job_id, "--env", env, "-o", "json"], hide_output=True)

    output_split = output.split("\n")

    for line in output_split:
        if line.startswith("[") or line.startswith("{"):
            return json.loads(line)

    return {}


def refresh(api_name: str, env: str, force: bool = False) -> dict:
    """
    Restart all of the replicas for a Realtime API without downtime.

    Args:
        api_name: API to refresh.
        env: Name of the environment hosting this API.
        force: Override the in-progress API update. Defaults to False.

    Returns:
        dict: The status of the refresh.
    """
    args = ["refresh", api_name, "--env", env, "-o", "json"]

    if force:
        args.append("--force")

    run_cli(args, hide_output=False)


def delete_api(api_name: str, env: str, keep_cache: bool = False):
    """
    Delete this API.

    Args:
        api_name: Name of the API to delete.
        env : Name of the environment hosting this API.
        keep_cache: Retain the cached data for this API. Defaults to False.
    """
    args = [
        "delete",
        api_name,
        "--env",
        env,
        "--force",
        "-o",
        "json",
    ]

    if keep_cache:
        args += "--keep-cache"

    run_cli(args)


def stop_job(api_name: str, job_id: str, env: str, keep_cache: bool = False):
    """
    Stop a running job.

    Args:
        api_name: Name of the Batch API.
        job_id: ID of the Job to stop.
        env : Name of the environment hosting this API.
        keep_cache: Retain the cached data for this API. Defaults to False.
    """
    args = [
        "delete",
        api_name,
        job_id,
        "--env",
        env,
        "--force",
        "-o",
        "json",
    ]

    if keep_cache:
        args += "--keep-cache"

    run_cli(args)


def logs_api(
    api_name: str,
    env: str,
    stdout_iterator: Callable[[str], bool] = None,
    hide_output: bool = False,
):
    """
    Stream the logs of an API.

    Args:
        api_name: Name of the API.
        env: Name of the environment hosting this API.
        stdout_iterator: A function that gets called on each character in the log stream and returns if the streaming should continue. Defaults to None.
        hide_output: Disable streaming the logs to stdout. Defaults to False.
    """
    run_cli(
        ["logs", api_name, "--env", env],
        hide_output=hide_output,
        stdout_iterator=stdout_iterator,
    )


def logs_job(
    api_name: str,
    job_id: str,
    env: str,
    stdout_iterator: Callable[[str], bool] = None,
    hide_output: bool = False,
):
    """
    Stream the logs of a Job.

    Args:
        api_name: Name of the Batch API.
        job_id: Job ID.
        env: Name of the environment hosting this Batch API.
        stdout_iterator: A function that gets called on each character in the log stream and returns if the streaming should continue. Defaults to None.
        hide_output: Disable streaming the logs to stdout. Defaults to False.
    """
    run_cli(
        ["logs", api_name, job_id, "--env", env],
        hide_output=hide_output,
        stdout_iterator=stdout_iterator,
    )


def env_configure_aws_provider(
    name: str,
    operator_endpoint: str,
    aws_access_key_id: str,
    aws_secret_access_key: str,
):
    """
    Create an environment to connect to an AWS Cortex cluster.

    Args:
        name: Name of the environment to create.
        operator_endpoint: The endpoint for the operator of your Cortex Cluster. You can get this endpoint using the Cortex CLI command `cortex cluster info -c cluster.yaml`.
        aws_access_key_id: AWS Credentials that will be used for authentication.
        aws_secret_access_key: AWS Credentials that will be used for authentication.
    """
    run_cli(
        [
            "env",
            "configure",
            name,
            "--provider",
            "aws",
            "--operator-endpoint",
            operator_endpoint,
            "--aws-access-key-id",
            aws_access_key_id,
            "--aws-secret-access-key",
            aws_secret_access_key,
        ],
        hide_output=True,
    )


def env_configure_local_provider(
    aws_access_key_id: Optional[str] = None,
    aws_secret_access_key: Optional[str] = None,
    aws_region: Optional[str] = None,
):
    """
    Configure your local environment with AWS credentials which will used to download models from S3, authenticate to ECR and be passed in to your local API deployments.

    Args:
        aws_access_key_id: AWS Credentials that will be used for authentication.
        aws_secret_access_key: AWS Credentials that will be used for authentication.
        aws_region: AWS region.
    """

    args = [
        "env",
        "configure",
        "--provider",
        "local",
    ]

    if aws_region is not None:
        args += ["--aws-region", aws_region]

    if aws_access_key_id is not None:
        args += ["--aws-access-key-id", aws_access_key_id]

    if aws_secret_access_key is not None:
        args += ["--aws-secret-access-key", aws_secret_access_key]

    run_cli(args, hide_output=True)


def env_list() -> list:
    """
    List all environments configured on this machine.
    """
    _, output = run_cli(["env", "list", "--output", "json"], hide_output=True)
    return json.loads(output)


def env_delete(name: str):
    """
    Delete an environment configured on this machine.

    Args:
        name (str): Name of the environment to delete.
    """
    run_cli(["env", "delete", name], hide_output=True)
