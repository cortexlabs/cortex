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

import subprocess
import sys

import yaml

from e2e.exceptions import ClusterCreationException, ClusterDeletionException


def create_cluster(cluster_config: str):
    """Create a cortex cluster from a cluster config"""
    with open(cluster_config) as f:
        config = yaml.safe_load(f)

    cluster_name = config["cluster_name"]
    provider = config["provider"]

    if provider == "gcp":
        cluster_cmd = "cluster-gcp"
    else:
        cluster_cmd = "cluster"

    p = subprocess.run(
        [
            "cortex",
            cluster_cmd,
            "up",
            cluster_config,
            "-y",
            "--configure-env",
            f"{cluster_name}-{provider}",
        ],
        stdout=sys.stdout,
        stderr=sys.stderr,
    )

    if p.returncode != 0:
        raise ClusterCreationException(f"failed to create cluster with config: {cluster_config}")


def delete_cluster(cluster_config: str):
    """Delete a cortex cluster from a cluster config"""
    with open(cluster_config) as f:
        config = yaml.safe_load(f)

    if config["provider"] == "gcp":
        cluster_cmd = "cluster-gcp"
    else:
        cluster_cmd = "cluster"

    p = subprocess.run(
        ["cortex", cluster_cmd, "down", "-y", "--config", cluster_config],
        stdout=sys.stdout,
        stderr=sys.stderr,
    )

    if p.returncode != 0:
        raise ClusterDeletionException(f"failed to delete cluster with config: {cluster_config}")
