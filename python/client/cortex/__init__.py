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
from typing import Optional, List

from cortex.binary import run_cli
from cortex.client import Client
from cortex.exceptions import NotFound
from cortex.telemetry import sentry_wrapper


__version__ = "0.38.0"  # CORTEX_VERSION


@sentry_wrapper
def client(env: Optional[str] = None) -> Client:
    """
    Initialize a client based on the specified environment. If no environment is specified, it will attempt to use the default environment.

    Args:
        env: Name of the environment to use.

    Returns:
        Cortex client that can be used to deploy and manage APIs in the specified environment.
    """
    environments = env_list()

    if env is None:
        if not environments.get("default_environment"):
            raise NotFound("no default environment configured")
        env = environments["default_environment"]

    found = False
    for environment in environments["environments"]:
        if environment["name"] == env:
            found = True
            break
    if not found:
        raise NotFound(f"can't find environment {env}, create one by calling `cortex.new_client()`")

    return Client(environment)


@sentry_wrapper
def new_client(
    name: str,
    operator_endpoint: str,
) -> Client:
    """
    Create a new environment to connect to an existing cluster, and initialize a client to deploy and manage APIs on that cluster.

    Args:
        name: Name of the environment to create.
        operator_endpoint: The endpoint for the operator of your Cortex cluster. You can get this endpoint by running the CLI command `cortex cluster info`.

    Returns:
        Cortex client that can be used to deploy and manage APIs on a cluster.
    """
    cli_args = [
        "env",
        "configure",
        name,
        "--operator-endpoint",
        operator_endpoint,
    ]

    run_cli(cli_args, hide_output=True)

    return client(name)


@sentry_wrapper
def env_list() -> List:
    """
    List all environments configured on this machine.
    """
    output = run_cli(["env", "list", "--output", "json"], hide_output=True)
    return json.loads(output.strip())


@sentry_wrapper
def env_delete(name: str):
    """
    Delete an environment configured on this machine.

    Args:
        name: Name of the environment to delete.
    """
    run_cli(["env", "delete", name], hide_output=True)
