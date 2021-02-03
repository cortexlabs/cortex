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
import time
import sys
import subprocess
import threading
import yaml
import uuid
import dill
import inspect
import shutil
from pathlib import Path

from typing import List, Dict, Optional, Tuple, Callable, Union
from cortex.binary import run_cli, get_cli_path
from cortex import util

# Change if PYTHONVERSION changes
EXPECTED_PYTHON_VERSION = "3.6.9"


def cli_config_dir() -> Path:
    cli_config_dir = os.environ.get("CORTEX_CLI_CONFIG_DIR", "")
    if cli_config_dir == "":
        return Path.home() / ".cortex"
    return Path(cli_config_dir).expanduser().resolve()


class Client:
    def __init__(self, env: dict):
        """
        A client to deploy and manage APIs in the specified environment.

        Args:
            env: Environment config
        """
        self.env = env
        self.env_name = env["name"]

    # CORTEX_VERSION_MINOR
    def create_api(
        self,
        api_spec: dict,
        predictor=None,
        task=None,
        requirements=[],
        conda_packages=[],
        project_dir: Optional[str] = None,
        force: bool = True,
        wait: bool = False,
    ) -> list:
        """
        Deploy an API.

        Args:
            api_spec: A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/0.28/ for schema.
            predictor: A Cortex Predictor class implementation. Not required for TaskAPI/TrafficSplitter kinds.
            task: A callable class/function implementation. Not required for RealtimeAPI/BatchAPI/TrafficSplitter kinds.
            requirements: A list of PyPI dependencies that will be installed before the predictor class implementation is invoked.
            conda_packages: A list of Conda dependencies that will be installed before the predictor class implementation is invoked.
            project_dir: Path to a python project.
            force: Override any in-progress api updates.
            wait: Streams logs until the APIs are ready.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """

        if self.env["provider"] == "gcp" and wait:
            raise ValueError(
                "`wait` flag is not supported for clusters on GCP, please set the `wait` flag to false"
            )

        if project_dir is not None:
            if predictor is not None:
                raise ValueError(
                    "`predictor` and `project_dir` parameters cannot be specified at the same time, please choose one"
                )
            if task is not None:
                raise ValueError(
                    "`task` and `project_dir` parameters cannot be specified at the same time, please choose one"
                )

        if project_dir is not None:
            cortex_yaml_path = os.path.join(project_dir, f".cortex-{uuid.uuid4()}.yaml")

            with util.open_temporarily(cortex_yaml_path, "w") as f:
                yaml.dump([api_spec], f)  # write a list
                return self._deploy(cortex_yaml_path, force, wait)

        api_kind = api_spec.get("kind")
        if api_kind == "TrafficSplitter":
            if predictor:
                raise ValueError(f"`predictor` parameter cannot be specified for {api_kind} kind")
            if task:
                raise ValueError(f"`task` parameter cannot be specified for {api_kind} kind")
        elif api_kind == "TaskAPI":
            if predictor:
                raise ValueError(f"`predictor` parameter cannnot be specified for {api_kind} kind")
            if task is None:
                raise ValueError(f"`task` parameter must be specified for {api_kind} kind")
        elif api_kind in ["BatchAPI", "RealtimeAPI"]:
            if not predictor:
                raise ValueError(f"`predictor` parameter must be specified for {api_kind}")
            if task:
                raise ValueError(f"`task` parameter cannot be specified for {api_kind}")
        else:
            raise ValueError(
                f"invalid {api_kind} kind, `api_spec` must have the `kind` field set to one of the following kinds: {['TrafficSplitter', 'TaskAPI', 'BatchAPI', 'RealtimeAPI']}"
            )

        if api_spec.get("name") is None:
            raise ValueError("`api_spec` must have the `name` key set")

        project_dir = cli_config_dir() / "deployments" / api_spec["name"]

        if project_dir.exists():
            shutil.rmtree(str(project_dir))

        project_dir.mkdir(parents=True)

        cortex_yaml_path = os.path.join(project_dir, "cortex.yaml")

        if api_kind == "TrafficSplitter":
            # for deploying a traffic splitter
            with open(cortex_yaml_path, "w") as f:
                yaml.dump([api_spec], f)  # write a list
                return self._deploy(cortex_yaml_path, force=force, wait=wait)

        actual_version = (
            f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}"
        )

        if actual_version != EXPECTED_PYTHON_VERSION:
            is_python_set = any(
                conda_dep.startswith("python=") or "::python=" in conda_dep
                for conda_dep in conda_packages
            )

            if not is_python_set:
                conda_packages = [f"python={actual_version}", "pip=19.*"] + conda_packages

        if len(requirements) > 0:
            with open(project_dir / "requirements.txt", "w") as requirements_file:
                requirements_file.write("\n".join(requirements))

        if len(conda_packages) > 0:
            with open(project_dir / "conda-packages.txt", "w") as conda_file:
                conda_file.write("\n".join(conda_packages))

        if api_kind in ["BatchAPI", "RealtimeAPI"]:
            if not inspect.isclass(predictor):
                raise ValueError("`predictor` parameter must be a class definition")

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

        if api_kind == "TaskAPI":
            if not callable(task):
                raise ValueError(
                    "`task` parameter must be a callable (e.g. a function definition or a class definition called `Task` with a `__call__` method implemented"
                )
            with open(project_dir / "task.pickle", "wb") as pickle_file:
                dill.dump(task, pickle_file)
                if api_spec.get("definition") is None:
                    api_spec["definition"] = {}
                api_spec["definition"]["path"] = "task.pickle"

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
            self.env_name,
            "-o",
            "mixed",
            "-y",
        ]

        if force:
            args.append("--force")

        output = run_cli(args, mixed_output=True)

        deploy_results = json.loads(output.strip())

        deploy_result = deploy_results[0]

        if not wait:
            return deploy_result

        # logging immediately will show previous versions of the replica terminating;
        # wait a few seconds for the new replicas to start initializing
        time.sleep(5)

        def stream_to_stdout(process):
            for c in iter(lambda: process.stdout.read(1), ""):
                sys.stdout.write(c)
                sys.stdout.flush()

        api_name = deploy_result["api"]["spec"]["name"]
        if deploy_result["api"]["spec"]["kind"] != "RealtimeAPI":
            return deploy_result

        env = os.environ.copy()
        env["CORTEX_CLI_INVOKER"] = "python"
        process = subprocess.Popen(
            [get_cli_path(), "logs", "--env", self.env_name, api_name, "-y"],
            stderr=subprocess.STDOUT,
            stdout=subprocess.PIPE,
            encoding="utf8",
            errors="replace",  # replace non-utf8 characters with `?` instead of failing
            env=env,
        )

        streamer = threading.Thread(target=stream_to_stdout, args=[process])
        streamer.start()

        while process.poll() is None:
            api = self.get_api(api_name)
            if api["status"]["status_code"] != "status_updating":
                time.sleep(5)  # accommodate latency in log streaming from the cluster
                process.terminate()
                break
            time.sleep(5)
        streamer.join(timeout=10)

        return api

    def get_api(self, api_name: str) -> dict:
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

    def list_apis(self) -> list:
        """
        List all APIs in the environment.

        Returns:
            List of APIs, including information such as the API specification, endpoint, status, and metrics (if applicable).
        """
        args = ["get", "-o", "json", "--env", self.env_name]

        output = run_cli(args, hide_output=True)

        return json.loads(output.strip())

    def get_job(self, api_name: str, job_id: str) -> dict:
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

    def patch(self, api_spec: dict, force: bool = False) -> dict:
        """
        Update the api specification for an API that has already been deployed.

        Args:
            api_spec: The new api specification to apply
            force: Override an already in-progress API update.
        """

        cortex_yaml_file = cli_config_dir() / "deployments" / f"cortex-{str(uuid.uuid4())}.yaml"
        with util.open_temporarily(cortex_yaml_file, "w") as f:
            yaml.dump([api_spec], f)
            args = ["patch", cortex_yaml_file, "--env", self.env_name, "-o", "json"]

            if force:
                args.append("--force")

            output = run_cli(args, hide_output=True)
            return json.loads(output.strip())

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
            self.env_name,
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

    def stream_api_logs(
        self,
        api_name: str,
    ):
        """
        Stream the logs of an API.

        Args:
            api_name: Name of the API.
        """
        args = ["logs", api_name, "--env", self.env_name, "-y"]
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
        args = ["logs", api_name, job_id, "--env", self.env_name, "-y"]
        run_cli(args)
