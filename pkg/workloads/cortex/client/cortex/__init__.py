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

from typing import List, Dict, Optional, Tuple, Callable, Union
import json

from cortex.client import Client
from cortex.binary import run_cli
from cortex.exceptions import NotFound


def client(env: str):
    """
    Initialize a client based on the specified environment.

    To deploy your API locally:
        import cortex
        c = cortex.client("local")
        c.deploy("./cortex.yaml")

    To deploy your API to a cluster:
        1. Spin up a cluster using the CLI command `cortex cluster up` in terminal.
           An environment named `aws` will be created for you by the CLI once the cluster has spun up.
        2. import cortex
           c = cortex.client("aws")
           c.deploy("./cortex.yaml")

    Args:
        env: Name of the environment to use to initialize the Cortex client.

    Returns:
        Cortex client that can be used to deploy and manage APIs in the specified environment.
    """
    environments = env_list()

    found = False

    for environment in environments:
        if environment["name"] == env:
            found = True

    if not found:
        raise NotFound(f"can't find environment {env}, create one by calling `new_cluster_client`")

    return Client(env)


def local_client(
    aws_access_key_id: str,
    aws_secret_access_key: str,
    aws_region: str,
) -> Client:
    """
    Initialize a client to deploy and manage APIs locally.
    The specified AWS credentials will used to download models from S3, authenticate to ECR and be passed in to your local API deployments.

    Args:
        aws_access_key_id: AWS Credentials that will be used for authentication.
        aws_secret_access_key: AWS Credentials that will be used for authentication.
        aws_region: AWS region.

    Returns:
        Cortex client that can be used to deploy and manage APIs locally.
    """
    args = [
        "env",
        "configure",
        "--provider",
        "local",
        "--aws-region",
        aws_region,
        "--aws-access-key-id",
        aws_access_key_id,
        "--aws-secret-access-key",
        aws_secret_access_key,
    ]

    run_cli(args, hide_output=True)

    return Client("local")


def new_cluster_client(
    name: str,
    operator_endpoint: str,
    aws_access_key_id: str,
    aws_secret_access_key: str,
) -> Client:
    """
    Create a new Initialize a client to deploy and manage APIs on a Cortex cluster.

    Args:
        name: Name of the environment to create.
        operator_endpoint: The endpoint for the operator of your Cortex Cluster. You can get this endpoint using the Cortex CLI command `cortex cluster info -c cluster.yaml`.
        aws_access_key_id: AWS Credentials that will be used for authentication.
        aws_secret_access_key: AWS Credentials that will be used for authentication.

    Returns:
        Cortex client that can be used to deploy and manage APIs on a Cortex Cluster.
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

    return Client(name)


def env_list() -> list:
    """
    List all environments configured on this machine.
    """
    output = run_cli(["env", "list", "--output", "json"], hide_output=True)
    return json.loads(output.strip())


def env_delete(name: str):
    """
    Delete an environment configured on this machine.
    Args:
        name: Name of the environment to delete.
    """
    run_cli(["env", "delete", name], hide_output=True)
