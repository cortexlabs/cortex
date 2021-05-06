#!/usr/bin/env bash

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
import sys
from pathlib import Path

NEURON_CORES_PER_INF = 4


def extract_from_handler(api_kind: str, handler_config: dict, compute_config: dict) -> dict:
    handler_type = handler_config["type"].lower()

    env_vars = {
        "CORTEX_LOG_LEVEL": handler_config["log_level"].upper(),
        "CORTEX_PROCESSES_PER_REPLICA": handler_config["processes_per_replica"],
        "CORTEX_THREADS_PER_PROCESS": handler_config["threads_per_process"],
        "CORTEX_DEPENDENCIES_PIP": handler_config["dependencies"]["pip"],
        "CORTEX_DEPENDENCIES_CONDA": handler_config["dependencies"]["conda"],
        "CORTEX_DEPENDENCIES_SHELL": handler_config["dependencies"]["shell"],
    }

    if handler_config.get("python_path") is not None:
        env_vars["CORTEX_PYTHON_PATH"] = os.path.normpath(
            os.path.join("/mnt", "project", handler_config["python_path"])
        )

    if api_kind == "RealtimeAPI":
        if handler_config.get("protobuf_path") is not None:
            env_vars["CORTEX_SERVING_PROTOCOL"] = "grpc"
            env_vars["CORTEX_PROTOBUF_FILE"] = os.path.join(
                "/mnt", "project", handler_config["protobuf_path"]
            )
        else:
            env_vars["CORTEX_SERVING_PROTOCOL"] = "http"

    if handler_type == "tensorflow":
        env_vars["CORTEX_TF_BASE_SERVING_PORT"] = "9000"
        env_vars["CORTEX_TF_SERVING_HOST"] = "localhost"

    if compute_config.get("inf", 0) > 0:
        if handler_type == "python":
            env_vars["NEURON_RTD_ADDRESS"] = "unix:/sock/neuron.sock"
            env_vars["NEURONCORE_GROUP_SIZES"] = int(
                compute_config["inf"]
                * NEURON_CORES_PER_INF
                / handler_config.get("processes_per_replica", 1)
            )

        if handler_type == "tensorflow":
            env_vars["CORTEX_MULTIPLE_TF_SERVERS"] = "yes"
            env_vars["CORTEX_ACTIVE_NEURON"] = "yes"

    return env_vars


def extract_from_task_definition(definition_config: dict, compute_config: dict) -> dict:
    env_vars = {
        "CORTEX_LOG_LEVEL": definition_config["log_level"].upper(),
        "CORTEX_DEPENDENCIES_PIP": definition_config["dependencies"]["pip"],
        "CORTEX_DEPENDENCIES_CONDA": definition_config["dependencies"]["conda"],
        "CORTEX_DEPENDENCIES_SHELL": definition_config["dependencies"]["shell"],
    }

    if definition_config.get("python_path") is not None:
        env_vars["CORTEX_PYTHON_PATH"] = os.path.normpath(
            os.path.join("/mnt", "project", definition_config["python_path"])
        )

    if compute_config.get("inf", 0) > 0:
        env_vars["NEURON_RTD_ADDRESS"] = "unix:/sock/neuron.sock"
        env_vars["NEURONCORE_GROUP_SIZES"] = int(compute_config["inf"] * NEURON_CORES_PER_INF)

    return env_vars


def extract_from_autoscaling(autoscaling_config: dict):
    return {"CORTEX_MAX_REPLICA_CONCURRENCY": autoscaling_config["max_replica_concurrency"]}


def set_env_vars_for_s6(env_vars: dict):
    s6_env_base = "/var/run/s6/container_environment"

    Path(s6_env_base).mkdir(parents=True, exist_ok=True)

    for k, v in env_vars.items():
        if v is not None:
            Path(f"{s6_env_base}/{k}").write_text(str(v))


def print_env_var_exports(env_vars: dict):
    for k, v in env_vars.items():
        if v is not None:
            print(f"export {k}='{v}'")


def main(api_spec_path: str):
    with open(api_spec_path, "r") as f:
        api_config = json.load(f)

    api_kind = api_config["kind"]
    env_vars = {
        "CORTEX_SERVING_PORT": 8888,
        "CORTEX_CACHE_DIR": "/mnt/cache",
        "CORTEX_PROJECT_DIR": "/mnt/project",
        "CORTEX_CLI_CONFIG_DIR": "/mnt/client",
        "CORTEX_MODEL_DIR": "/mnt/model",
        "CORTEX_LOG_CONFIG_FILE": "/src/cortex/serve/log_config.yaml",
        "CORTEX_PYTHON_PATH": "/mnt/project",
        "HOST_IP": os.environ.get("HOST_IP", "localhost"),
        "CORTEX_KIND": api_kind,
    }

    handler_config = api_config.get("handler", None)
    compute_config = api_config.get("compute", None)
    definition_config = api_config.get("definition", None)
    autoscaling_config = api_config.get("autoscaling", None)

    if handler_config is not None:
        env_vars.update(extract_from_handler(api_kind, handler_config, compute_config))

    if definition_config is not None:
        env_vars.update(extract_from_task_definition(definition_config, compute_config))

    if autoscaling_config is not None:
        env_vars.update(extract_from_autoscaling(autoscaling_config))

    set_env_vars_for_s6(env_vars)
    print_env_var_exports(env_vars)


if __name__ == "__main__":
    main(sys.argv[1])
