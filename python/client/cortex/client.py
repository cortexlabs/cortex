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
import os
import shutil
import subprocess
import sys
import threading
import time
import yaml
from pathlib import Path
from typing import Optional, List, Dict, Any

from cortex import util
from cortex.binary import run_cli, get_cli_path
from cortex.telemetry import sentry_wrapper


class Client:
    @sentry_wrapper
    def __init__(self, env: Dict):
        """
        A client to deploy and manage APIs in the specified environment.

        Args:
            env: Environment config
        """

        self.env = env
        self.env_name = env["name"]

    # CORTEX_VERSION_MINOR
    @sentry_wrapper
    def deploy(
        self,
        api_spec: Dict[str, Any],
        force: bool = True,
        wait: bool = False,
    ):
        """
        Deploy or update an API.

        Args:
            api_spec: A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/0.38/ for schema.
            force: Override any in-progress api updates.
            wait: Block until the API is ready.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """

        temp_deploy_dir = util.cli_config_dir() / "deployments" / api_spec["name"]
        if temp_deploy_dir.exists():
            shutil.rmtree(str(temp_deploy_dir))
        temp_deploy_dir.mkdir(parents=True)

        cortex_yaml_path = os.path.join(temp_deploy_dir, "cortex.yaml")

        with util.open_temporarily(cortex_yaml_path, "w", delete_parent_if_empty=True) as f:
            yaml.dump([api_spec], f)  # write a list
            return self.deploy_from_file(cortex_yaml_path, force=force, wait=wait)

    # CORTEX_VERSION_MINOR
    @sentry_wrapper
    def deploy_from_file(
        self,
        config_file: str,
        force: bool = False,
        wait: bool = False,
    ) -> Dict:
        """
        Deploy or update APIs specified in a configuration file.

        Args:
            config_file: Local path to a yaml file defining Cortex API(s). See https://docs.cortex.dev/v/0.38/ for schema.
            force: Override any in-progress api updates.
            wait: Block until the API is ready.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """

        args = [
            "deploy",
            config_file,
            "--env",
            self.env_name,
            "-o",
            "json",
            "-y",
        ]

        if force:
            args.append("--force")

        output = run_cli(args, hide_output=True)

        deploy_results = json.loads(output.strip())

        deploy_result = deploy_results[0]

        if not wait:
            return deploy_result

        api_name = deploy_result["api"]["spec"]["name"]
        if (
            deploy_result["api"]["spec"]["kind"] != "RealtimeAPI"
            and deploy_result["api"]["spec"]["kind"] != "AsyncAPI"
        ):
            return deploy_result

        while True:
            time.sleep(5)
            api = self.get_api(api_name)
            if api["status"]["status_code"] != "status_updating":
                break

        return api

    @sentry_wrapper
    def get_api(self, api_name: str) -> Dict:
        """
        Get information about an API.

        Args:
            api_name: Name of the API.

        Returns:
            Information about the API, including the API specification, endpoint, status, and metrics (if applicable).
        """
        output = run_cli(["get", api_name, "--env", self.env_name, "-o", "json"], hide_output=True)

        apis = json.loads(output.strip())
        return apis[0]

    @sentry_wrapper
    def list_apis(self) -> List:
        """
        List all APIs in the environment.

        Returns:
            List of APIs, including information such as the API specification, endpoint, status, and metrics (if applicable).
        """
        args = ["get", "-o", "json", "--env", self.env_name]

        output = run_cli(args, hide_output=True)

        return json.loads(output.strip())

    @sentry_wrapper
    def get_job(self, api_name: str, job_id: str) -> Dict:
        """
        Get information about a submitted job.

        Args:
            api_name: Name of the Batch/Task API.
            job_id: Job ID.

        Returns:
            Information about the job, including the job status, worker status, and job progress.
        """
        args = ["get", api_name, job_id, "--env", self.env_name, "-o", "json"]

        output = run_cli(args, hide_output=True)

        return json.loads(output.strip())

    @sentry_wrapper
    def refresh(self, api_name: str, force: bool = False):
        """
        Restart all of the replicas for a Realtime API without downtime.

        Args:
            api_name: Name of the API to refresh.
            force: Override an already in-progress API update.
        """
        args = ["refresh", api_name, "--env", self.env_name, "-o", "json"]

        if force:
            args.append("--force")

        run_cli(args, hide_output=True)

    @sentry_wrapper
    def delete(self, api_name: str, keep_cache: bool = False):
        """
        Delete an API.

        Args:
            api_name: Name of the API to delete.
            keep_cache: Whether to retain the cached data for this API.
        """
        args = [
            "delete",
            api_name,
            "--env",
            self.env_name,
            "--force",
            "-o",
            "json",
        ]

        if keep_cache:
            args.append("--keep-cache")

        run_cli(args, hide_output=True)

    @sentry_wrapper
    def stop_job(self, api_name: str, job_id: str, keep_cache: bool = False):
        """
        Stop a running job.

        Args:
            api_name: Name of the Batch/Task API.
            job_id: ID of the Job to stop.
        """
        args = [
            "delete",
            api_name,
            job_id,
            "--env",
            self.env_name,
            "-o",
            "json",
        ]

        run_cli(args)
