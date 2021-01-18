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

from typing import List, Dict, Optional, Tuple, Callable, Union
import json

from cortex.client import Client
from cortex.binary import run_cli
from cortex.exceptions import NotFound


def client(env: str) -> Client:
    """
    Initialize a client based on the specified environment.

    Args:
        env: Name of the environment to use.

    Returns:
        Cortex client that can be used to deploy and manage APIs in the specified environment.
    """
    environments = env_list()

    found = False

    for environment in environments:
        if environment["name"] == env:
            found = True
            break

    if not found:
        raise NotFound(
            f"can't find environment {env}, create one by calling `cortex.cluster_client()`"
        )

    return Client(environment)


def cluster_client(
    name: str,
    provider: str,
    operator_endpoint: str,
    aws_access_key_id: Optional[str] = None,
    aws_secret_access_key: Optional[str] = None,
) -> Client:
    """
    Create a new environment to connect to an existing Cortex Cluster, and initialize a client to deploy and manage APIs on that cluster.

    Args:
        name: Name of the environment to create.
        provider: The provider of your Cortex cluster. Can be "aws" or "gcp".
        operator_endpoint: The endpoint for the operator of your Cortex Cluster. You can get this endpoint by running the CLI command `cortex cluster info` for an AWS provider or `cortex cluster-gcp info` for a GCP provider.
        aws_access_key_id: AWS access key ID. Required when `provider` is set to "aws".
        aws_secret_access_key: AWS secret access key. Required when `provider` is set to "aws".

    Returns:
        Cortex client that can be used to deploy and manage APIs on a Cortex Cluster.
    """
    cli_args = [
        "env",
        "configure",
        name,
        "--provider",
        provider,
        "--operator-endpoint",
        operator_endpoint,
    ]
    if provider == "aws":
        cli_args += [
            "--aws-access-key-id",
            aws_access_key_id,
            "--aws-secret-access-key",
            aws_secret_access_key,
        ]
    run_cli(cli_args, hide_output=True)

    return client(name)


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
