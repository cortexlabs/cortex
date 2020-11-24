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
import yaml
import uuid
import dill
import inspect
from pathlib import Path

from typing import List, Dict, Optional, Tuple, Callable, Union
from cortex.binary import run_cli, get_cli_path
from cortex import util


class Client:
    def __init__(self, env: str):
        """
        A client to deploy and manage APIs in the specified environment.

        Args:
            env: Name of the environment to use.
        """
        self.env = env

    # CORTEX_VERSION_MINOR x5
    def deploy(
        self,
        api_spec: dict,
        predictor=None,
        pip_dependencies=[],
        conda_dependencies=[],
        project_dir: Optional[str] = None,
        force: bool = False,
        wait: bool = False,
    ) -> list:
        """
        Deploy an API.

        Args:
            api_spec: A dictionary defining a single Cortex API. Schema can be found here:
                → Realtime API: https://docs.cortex.dev/v/0.23/deployments/realtime-api/api-configuration
                → Batch API: https://docs.cortex.dev/v/0.23/deployments/batch-api/api-configuration
                → Traffic Splitter: https://docs.cortex.dev/v/0.23/deployments/realtime-api/traffic-splitter
            predictor: A Cortex Predictor class implementation. Not required when deploying a traffic splitter.
                → Realtime API: https://docs.cortex.dev/v/0.23/deployments/realtime-api/predictors
                → Batch API: https://docs.cortex.dev/v/0.23/deployments/batch-api/predictors
            pip_dependencies: A list of PyPI dependencies that will be installed before the predictor class implementation is invoked.
            conda_dependencies: A list of Conda dependencies that will be installed before the predictor class implementation is invoked.
            project_dir: Path to a python project.
            force: Override any in-progress api updates.
            wait: Streams logs until the APIs are ready.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """

        if project_dir is not None and predictor is not None:
            raise ValueError(
                "`predictor` and `project_dir` parameters cannot be specified at the same time, please choose one"
            )

        if project_dir is not None:
            cortex_yaml_path = os.path.join(project_dir, f".cortex-{uuid.uuid4()}.yaml")

            with util.open_temporarily(cortex_yaml_path, "w") as f:
                yaml.dump([api_spec], f)  # write a list
                return self._deploy(cortex_yaml_path, force, wait)

        project_dir = Path.home() / ".cortex" / "deployments" / str(uuid.uuid4())
        with util.open_tempdir(str(project_dir)):
            cortex_yaml_path = os.path.join(project_dir, "cortex.yaml")

            if predictor is None:
                # for deploying a traffic splitter
                with open(cortex_yaml_path, "w") as f:
                    yaml.dump([api_spec], f)  # write a list
                    return self._deploy(cortex_yaml_path, force=force, wait=wait)

            # Change if PYTHONVERSION changes
            expected_version = "3.6"
            actual_version = f"{sys.version_info.major}.{sys.version_info.minor}"
            if actual_version < expected_version:
                raise Exception("cortex is only supported for python versions >= 3.6")  # unexpected
            if actual_version > expected_version:
                is_python_set = any(
                    conda_dep.startswith("python=") or "::python=" in conda_dep
                    for conda_dep in conda_dependencies
                )

                if not is_python_set:
                    conda_dependencies = [
                        f"conda-forge::python={sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}"
                    ] + conda_dependencies

            if len(pip_dependencies) > 0:
                with open(project_dir / "requirements.txt", "w") as requirements_file:
                    requirements_file.write("\n".join(pip_dependencies))

            if len(conda_dependencies) > 0:
                with open(project_dir / "conda-packages.txt", "w") as conda_file:
                    conda_file.write("\n".join(conda_dependencies))

            if not inspect.isclass(predictor):
                raise ValueError("predictor parameter must be a class definition")

            with open(project_dir / "predictor.pickle", "wb") as pickle_file:
                dill.dump(predictor, pickle_file)
                if api_spec.get("predictor") is None:
                    api_spec["predictor"] = {}

                if predictor.__name__ == "PythonPredictor":
                    predictor_type = "python"
                if predictor.__name__ == "TensorFlowPredictor":
                    predictor_type = "tensorflow"
                if predictor.__name__ == "ONNXPredictor":
                    predictor_type = "onnx"

                api_spec["predictor"]["path"] = "predictor.pickle"
                api_spec["predictor"]["type"] = predictor_type

            with open(cortex_yaml_path, "w") as f:
                yaml.dump([api_spec], f)  # write a list
                return self._deploy(cortex_yaml_path, force=force, wait=wait)

    def _deploy(
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
