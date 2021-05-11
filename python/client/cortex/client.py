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

import inspect
import json
import os
import shutil
import subprocess
import sys
import threading
import time
import uuid
from pathlib import Path
from typing import Optional, List, Dict, Any

import dill
import yaml

from cortex import util
from cortex.binary import run_cli, get_cli_path
from cortex.consts import EXPECTED_PYTHON_VERSION
from cortex.telemetry import sentry_wrapper
from cortex.exceptions import InvalidKindForMethod


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
    def deploy(
        self,
        api_spec: Dict[str, Any],
        project_dir: str,
        force: bool = True,
        wait: bool = False,
    ):
        """
        Deploy API(s) from a project directory.

        Args:
            api_spec: A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/0.35/ for schema.
            project_dir: Path to a python project.
            force: Override any in-progress api updates.
            wait: Streams logs until the APIs are ready.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """
        return self._create_api(
            api_spec=api_spec,
            project_dir=project_dir,
            force=force,
            wait=wait,
        )

    # CORTEX_VERSION_MINOR
    def deploy_realtime_api(
        self,
        api_spec: Dict[str, Any],
        handler,
        requirements: Optional[List] = None,
        conda_packages: Optional[List] = None,
        force: bool = True,
        wait: bool = False,
    ) -> Dict:
        """
        Deploy a Realtime API.

        Args:
            api_spec: A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/0.35/workloads/realtime-apis/configuration for schema.
            handler: A Cortex Handler class implementation.
            requirements: A list of PyPI dependencies that will be installed before the handler class implementation is invoked.
            conda_packages: A list of Conda dependencies that will be installed before the handler class implementation is invoked.
            force: Override any in-progress api updates.
            wait: Streams logs until the APIs are ready.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """
        kind = api_spec.get("kind")
        if kind != "RealtimeAPI":
            raise InvalidKindForMethod(
                f"expected an api_spec with kind 'RealtimeAPI', got kind '{kind}' instead"
            )

        return self._create_api(
            api_spec=api_spec,
            handler=handler,
            requirements=requirements,
            conda_packages=conda_packages,
            force=force,
            wait=wait,
        )

    # CORTEX_VERSION_MINOR
    def deploy_async_api(
        self,
        api_spec: Dict[str, Any],
        handler,
        requirements: Optional[List] = None,
        conda_packages: Optional[List] = None,
        force: bool = True,
    ) -> Dict:
        """
        Deploy an Async API.

        Args:
            api_spec: A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/0.35/workloads/async-apis/configuration for schema.
            handler: A Cortex Handler class implementation.
            requirements: A list of PyPI dependencies that will be installed before the handler class implementation is invoked.
            conda_packages: A list of Conda dependencies that will be installed before the handler class implementation is invoked.
            force: Override any in-progress api updates.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """
        kind = api_spec.get("kind")
        if kind != "AsyncAPI":
            raise InvalidKindForMethod(
                f"expected an api_spec with kind 'AsyncAPI', got kind '{kind}' instead"
            )

        return self._create_api(
            api_spec=api_spec,
            handler=handler,
            requirements=requirements,
            conda_packages=conda_packages,
            force=force,
        )

    # CORTEX_VERSION_MINOR
    def deploy_batch_api(
        self,
        api_spec: Dict[str, Any],
        handler,
        requirements: Optional[List] = None,
        conda_packages: Optional[List] = None,
    ) -> Dict:
        """
        Deploy a Batch API.

        Args:
            api_spec: A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/0.35/workloads/batch-apis/configuration for schema.
            handler: A Cortex Handler class implementation.
            requirements: A list of PyPI dependencies that will be installed before the handler class implementation is invoked.
            conda_packages: A list of Conda dependencies that will be installed before the handler class implementation is invoked.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """

        kind = api_spec.get("kind")
        if kind != "BatchAPI":
            raise InvalidKindForMethod(
                f"expected an api_spec with kind 'BatchAPI', got kind '{kind}' instead"
            )

        return self._create_api(
            api_spec=api_spec,
            handler=handler,
            requirements=requirements,
            conda_packages=conda_packages,
        )

    # CORTEX_VERSION_MINOR
    def deploy_task_api(
        self,
        api_spec: Dict[str, Any],
        task,
        requirements: Optional[List] = None,
        conda_packages: Optional[List] = None,
    ) -> Dict:
        """
        Deploy a Task API.

        Args:
            api_spec: A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/0.35/workloads/task-apis/configuration for schema.
            task: A callable class implementation.
            requirements: A list of PyPI dependencies that will be installed before the handler class implementation is invoked.
            conda_packages: A list of Conda dependencies that will be installed before the handler class implementation is invoked.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """
        kind = api_spec.get("kind")
        if kind != "TaskAPI":
            raise InvalidKindForMethod(
                f"expected an api_spec with kind 'TaskAPI', got kind '{kind}' instead"
            )

        return self._create_api(
            api_spec=api_spec,
            task=task,
            requirements=requirements,
            conda_packages=conda_packages,
        )

    # CORTEX_VERSION_MINOR
    def deploy_traffic_splitter(
        self,
        api_spec: Dict[str, Any],
    ) -> Dict:
        """
        Deploy a Task API.

        Args:
            api_spec: A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/0.35/workloads/realtime-apis/traffic-splitter/configuration for schema.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """
        kind = api_spec.get("kind")
        if kind != "TrafficSplitter":
            raise InvalidKindForMethod(
                f"expected an api_spec with kind 'TrafficSplitter', got kind '{kind}' instead"
            )

        return self._create_api(
            api_spec=api_spec,
        )

    # CORTEX_VERSION_MINOR
    @sentry_wrapper
    def _create_api(
        self,
        api_spec: Dict,
        handler=None,
        task=None,
        requirements: Optional[List] = None,
        conda_packages: Optional[List] = None,
        project_dir: Optional[str] = None,
        force: bool = True,
        wait: bool = False,
    ) -> Dict:
        """
        Deploy an API.

        Args:
            api_spec: A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/0.35/ for schema.
            handler: A Cortex Handler class implementation. Not required for TaskAPI/TrafficSplitter kinds.
            task: A callable class/function implementation. Not required for RealtimeAPI/BatchAPI/TrafficSplitter kinds.
            requirements: A list of PyPI dependencies that will be installed before the handler class implementation is invoked.
            conda_packages: A list of Conda dependencies that will be installed before the handler class implementation is invoked.
            project_dir: Path to a python project.
            force: Override any in-progress api updates.
            wait: Streams logs until the APIs are ready.

        Returns:
            Deployment status, API specification, and endpoint for each API.
        """

        requirements = requirements if requirements is not None else []
        conda_packages = conda_packages if conda_packages is not None else []

        if project_dir is not None:
            if handler is not None:
                raise ValueError(
                    "`handler` and `project_dir` parameters cannot be specified at the same time, please choose one"
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
            if handler:
                raise ValueError(f"`handler` parameter cannot be specified for {api_kind} kind")
            if task:
                raise ValueError(f"`task` parameter cannot be specified for {api_kind} kind")
        elif api_kind == "TaskAPI":
            if handler:
                raise ValueError(f"`handler` parameter cannnot be specified for {api_kind} kind")
            if task is None:
                raise ValueError(f"`task` parameter must be specified for {api_kind} kind")
        elif api_kind in ["BatchAPI", "RealtimeAPI"]:
            if not handler:
                raise ValueError(f"`handler` parameter must be specified for {api_kind}")
            if task:
                raise ValueError(f"`task` parameter cannot be specified for {api_kind}")
        else:
            raise ValueError(
                f"invalid {api_kind} kind, `api_spec` must have the `kind` field set to one of the following kinds: "
                f"{['TrafficSplitter', 'TaskAPI', 'BatchAPI', 'RealtimeAPI']}"
            )

        if api_spec.get("name") is None:
            raise ValueError("`api_spec` must have the `name` key set")

        project_dir = util.cli_config_dir() / "deployments" / api_spec["name"]

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
            if not inspect.isclass(handler):
                raise ValueError("`handler` parameter must be a class definition")

            if api_spec.get("handler") is None:
                raise ValueError("`api_spec` must have the `handler` section defined")

            if api_spec["handler"].get("type") is None:
                raise ValueError(
                    "the `type` field in the `handler` section of the `api_spec` must be set (tensorflow or python)"
                )

            impl_rel_path = self._save_impl(handler, project_dir, "handler")
            api_spec["handler"]["path"] = impl_rel_path

        if api_kind == "TaskAPI":
            if not callable(task):
                raise ValueError(
                    "`task` parameter must be a callable (e.g. a function definition or a class definition called "
                    "`Task` with a `__call__` method implemented "
                )

            impl_rel_path = self._save_impl(task, project_dir, "task")
            if api_spec.get("definition") is None:
                api_spec["definition"] = {}
            api_spec["definition"]["path"] = impl_rel_path

        with open(cortex_yaml_path, "w") as f:
            yaml.dump([api_spec], f)  # write a list
            return self._deploy(cortex_yaml_path, force=force, wait=wait)

    def _save_impl(self, impl, project_dir: Path, filename: str) -> str:
        import __main__ as main

        is_interactive = not hasattr(main, "__file__")

        if is_interactive and impl.__module__ == "__main__":
            # class is defined in a REPL (e.g. jupyter)
            filename += ".pickle"
            with open(project_dir / filename, "wb") as pickle_file:
                dill.dump(impl, pickle_file)
                return filename

        filename += ".py"
        with open(project_dir / filename, "w") as f:
            f.write(dill.source.importable(impl, source=True))
            return filename

    def _deploy(
        self,
        config_file: str,
        force: bool = False,
        wait: bool = False,
    ) -> Dict:
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
    def patch(self, api_spec: Dict, force: bool = False) -> Dict:
        """
        Update the api specification for an API that has already been deployed.

        Args:
            api_spec: The new api specification to apply
            force: Override an already in-progress API update.
        """

        cortex_yaml_file = (
            util.cli_config_dir() / "deployments" / f"cortex-{str(uuid.uuid4())}.yaml"
        )
        with util.open_temporarily(cortex_yaml_file, "w") as f:
            yaml.dump([api_spec], f)
            args = ["patch", cortex_yaml_file, "--env", self.env_name, "-o", "json"]

            if force:
                args.append("--force")

            output = run_cli(args, hide_output=True)
            return json.loads(output.strip())

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

    @sentry_wrapper
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

    @sentry_wrapper
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
