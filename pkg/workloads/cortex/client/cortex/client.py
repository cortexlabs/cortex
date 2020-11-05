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
import os
import time
import sys
import subprocess
import threading

from typing import List, Dict, Optional, Tuple, Callable, Union
from cortex.binary import run_cli, get_cli_path


class Client:
    def __init__(self, env: str):
        """
        A client to deploy and manage APIs in the specified environment.

        Args:
            env: Name of the environment to use.
        """
        self.env = env

    def deploy(
        self,
        config_file: str,
        force: bool = False,
        wait: bool = False,
    ) -> list:
        """
        Deploy or update APIs specified in the config_file.

        Args:
            config_file: Local path to a yaml file defining Cortex APIs.
            force: Override any in-progress api updates.
            wait: Streams logs until the APIs are ready.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """

        args = [
            "deploy",
            config_file,
            "--env",
            self.env,
            "-o",
            "mixed",
        ]

        if force:
            args.append("--force")

        output = run_cli(args, mixed_output=True)

        deploy_results = json.loads(output.strip())

        if not wait:
            return deploy_results

        def stream_to_stdout(process):
            for c in iter(lambda: process.stdout.read(1), ""):
                sys.stdout.write(c)

        for deploy_result in deploy_results:
            api_name = deploy_result["api"]["spec"]["name"]
            kind = deploy_result["api"]["spec"]["kind"]
            if kind != "RealtimeAPI":
                continue

            env = os.environ.copy()
            env["CORTEX_CLI_INVOKER"] = "python"
            process = subprocess.Popen(
                [get_cli_path(), "logs", "--env", self.env, api_name],
                stderr=subprocess.STDOUT,
                stdout=subprocess.PIPE,
                encoding="utf8",
                env=env,
            )

            streamer = threading.Thread(target=stream_to_stdout, args=[process])
            streamer.start()

            while process.poll() is None:
                api = self.get_api(api_name)
                if api["status"]["status_code"] != "status_updating":
                    if api["status"]["status_code"] == "status_live":
                        time.sleep(2)
                    process.terminate()
                    break
                time.sleep(2)

        return deploy_results

    def get_api(self, api_name: str) -> dict:
        """
        Get information about an API.

        Args:
            api_name: Name of the API.

        Returns:
            Information about the API, including the API specification, endpoint, status, and metrics (if applicable).
        """
        output = run_cli(["get", api_name, "--env", self.env, "-o", "json"], hide_output=True)

        apis = json.loads(output.strip())
        return apis[0]

    def list_apis(self) -> list:
        """
        List all APIs in the environment.

        Returns:
            List of APIs, including information such as the API specification, endpoint, status, and metrics (if applicable).
        """
        args = ["get", "-o", "json", "--env", self.env]

        output = run_cli(args, hide_output=True)

        return json.loads(output.strip())

    def get_job(self, api_name: str, job_id: str) -> dict:
        """
        Get information about a submitted job.

        Args:
            api_name: Name of the Batch API.
            job_id: Job ID.

        Returns:
            Information about the job, including the job status, worker status, and job progress.
        """
        args = ["get", api_name, job_id, "--env", self.env, "-o", "json"]

        output = run_cli(args, hide_output=True)

        return json.loads(output.strip())

    def refresh(self, api_name: str, force: bool = False):
        """
        Restart all of the replicas for a Realtime API without downtime.

        Args:
            api_name: Name of the API to refresh.
            force: Override an already in-progress API update.
        """
        args = ["refresh", api_name, "--env", self.env, "-o", "json"]

        if force:
            args.append("--force")

        run_cli(args, hide_output=True)

    def delete_api(self, api_name: str, keep_cache: bool = False):
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
            self.env,
            "--force",
            "-o",
            "json",
        ]

        if keep_cache:
            args.append("--keep-cache")

        run_cli(args, hide_output=True)

    def stop_job(self, api_name: str, job_id: str, keep_cache: bool = False):
        """
        Stop a running job.

        Args:
            api_name: Name of the Batch API.
            job_id: ID of the Job to stop.
        """
        args = [
            "delete",
            api_name,
            job_id,
            "--env",
            self.env,
            "-o",
            "json",
        ]

        run_cli(args)

    def stream_api_logs(
        self,
        api_name: str,
    ):
        """
        Stream the logs of an API.

        Args:
            api_name: Name of the API.
        """
        args = ["logs", api_name, "--env", self.env]
        run_cli(args)

    def stream_job_logs(
        self,
        api_name: str,
        job_id: str,
    ):
        """
        Stream the logs of a Job.

        Args:
            api_name: Name of the Batch API.
            job_id: Job ID.
        """
        args = ["logs", api_name, job_id, "--env", self.env]
        run_cli(args)
