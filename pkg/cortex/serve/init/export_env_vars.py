import json
import os
import sys
from pathlib import Path

NEURON_CORES_PER_INF = 4


def extract_from_predictor(api_kind: str, predictor_config: dict, compute_config: dict) -> dict:
    predictor_type = predictor_config["type"].lower()

    env_vars = {
        "CORTEX_LOG_LEVEL": predictor_config["log_level"].upper(),
        "CORTEX_PROCESSES_PER_REPLICA": predictor_config["processes_per_replica"],
        "CORTEX_THREADS_PER_PROCESS": predictor_config["threads_per_process"],
        "CORTEX_DEPENDENCIES_PIP": predictor_config["dependencies"]["pip"],
        "CORTEX_DEPENDENCIES_CONDA": predictor_config["dependencies"]["conda"],
        "CORTEX_DEPENDENCIES_SHELL": predictor_config["dependencies"]["shell"],
    }

    if predictor_config.get("python_path") is not None:
        env_vars["CORTEX_PYTHON_PATH"] = os.path.normpath(
            os.path.join("/mnt", "project", predictor_config["python_path"])
        )

    if api_kind == "RealtimeAPI":
        if predictor_config.get("protobuf_path") is not None:
            env_vars["CORTEX_SERVING_PROTOCOL"] = "grpc"
            env_vars["CORTEX_PROTOBUF_FILE"] = os.path.join(
                "/mnt", "project", predictor_config["protobuf_path"]
            )
        else:
            env_vars["CORTEX_SERVING_PROTOCOL"] = "http"

    if compute_config.get("inf", 0) > 0:
        if predictor_type == "python":
            env_vars["NEURON_RTD_ADDRESS"] = "unix:/sock/neuron.sock"
            env_vars["NEURONCORE_GROUP_SIZES"] = int(
                compute_config["inf"]
                * NEURON_CORES_PER_INF
                / predictor_config.get("processes_per_replica", 1)
            )

        if predictor_type == "tensorflow":
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

    if compute_config is not None and compute_config.get("inf", 0) > 0:
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
            print(f"export {k}={v}")


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
        "HOST_IP": "localhost",
        "CORTEX_KIND": api_kind,
    }

    predictor_config = api_config.get("predictor", None)
    compute_config = api_config.get("compute", None)
    definition_config = api_config.get("definition", None)
    autoscaling_config = api_config.get("autoscaling", None)

    if predictor_config is not None:
        env_vars.update(extract_from_predictor(api_kind, predictor_config, compute_config))

    if definition_config is not None:
        env_vars.update(extract_from_task_definition(definition_config, compute_config))

    if autoscaling_config is not None:
        env_vars.update(extract_from_autoscaling(autoscaling_config))

    set_env_vars_for_s6(env_vars)
    print_env_var_exports(env_vars)


if __name__ == "__main__":
    main(sys.argv[1])