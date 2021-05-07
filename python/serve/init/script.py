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
import time

from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.telemetry import get_default_tags, init_sentry

init_sentry(tags=get_default_tags())
logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])

from cortex_internal.lib.type import (
    handler_type_from_api_spec,
    PythonHandlerType,
    TensorFlowHandlerType,
    TensorFlowNeuronHandlerType,
)
from cortex_internal.lib.model import (
    FileBasedModelsTreeUpdater,  # only when num workers > 1
    TFSModelLoader,
)
from cortex_internal.lib.checkers.pod import wait_neuron_rtd


def prepare_tfs_servers_api(api_spec: dict, model_dir: str) -> TFSModelLoader:
    # get TFS address-specific details
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")
    tf_base_serving_port = int(os.getenv("CORTEX_TF_BASE_SERVING_PORT", "9000"))

    # determine if multiple TF processes are required
    num_processes = 1
    has_multiple_tf_servers = os.getenv("CORTEX_MULTIPLE_TF_SERVERS")
    if has_multiple_tf_servers:
        num_processes = int(os.environ["CORTEX_PROCESSES_PER_REPLICA"])

    # initialize models for each TF process
    addresses = []
    for w in range(int(num_processes)):
        addresses.append(f"{tf_serving_host}:{tf_base_serving_port+w}")

    if len(addresses) == 1:
        return TFSModelLoader(
            interval=10,
            api_spec=api_spec,
            address=addresses[0],
            tfs_model_dir=model_dir,
            download_dir=model_dir,
        )
    return TFSModelLoader(
        interval=10,
        api_spec=api_spec,
        addresses=addresses,
        tfs_model_dir=model_dir,
        download_dir=model_dir,
    )


def are_models_specified(api_spec: dict) -> bool:
    """
    Checks if models have been specified in the API spec (cortex.yaml).

    Args:
        api_spec: API configuration.
    """
    handler_type = handler_type_from_api_spec(api_spec)

    if handler_type == PythonHandlerType and api_spec["handler"]["multi_model_reloading"]:
        models = api_spec["handler"]["multi_model_reloading"]
    elif handler_type != PythonHandlerType:
        models = api_spec["handler"]["models"]
    else:
        return False

    return models is not None


def is_model_caching_enabled(api_spec: dir) -> bool:
    handler_type = handler_type_from_api_spec(api_spec)

    if handler_type == PythonHandlerType and api_spec["handler"]["multi_model_reloading"]:
        models = api_spec["handler"]["multi_model_reloading"]
    elif handler_type != PythonHandlerType:
        models = api_spec["handler"]["models"]
    else:
        return False

    return models and models["cache_size"] and models["disk_cache_size"]


def main():
    # wait until neuron-rtd sidecar is ready
    uses_inferentia = os.getenv("CORTEX_ACTIVE_NEURON")
    if uses_inferentia:
        wait_neuron_rtd()

    # strictly for Inferentia
    has_multiple_tf_servers = os.getenv("CORTEX_MULTIPLE_TF_SERVERS")
    num_processes = int(os.environ["CORTEX_PROCESSES_PER_REPLICA"])
    if has_multiple_tf_servers:
        base_serving_port = int(os.environ["CORTEX_TF_BASE_SERVING_PORT"])
        used_ports = {}
        for w in range(int(num_processes)):
            used_ports[str(base_serving_port + w)] = False
        with open("/run/used_ports.json", "w+") as f:
            json.dump(used_ports, f)

    # get API spec
    spec_path = os.environ["CORTEX_API_SPEC"]
    cache_dir = os.getenv("CORTEX_CACHE_DIR")
    region = os.getenv("AWS_DEFAULT_REGION")  # when it's deployed to AWS

    with open(spec_path) as json_file:
        api_spec = json.load(json_file)

    handler_type = handler_type_from_api_spec(api_spec)
    caching_enabled = is_model_caching_enabled(api_spec)
    model_dir = os.getenv("CORTEX_MODEL_DIR")

    # start live-reloading when model caching not enabled > 1
    cron = None
    if not caching_enabled:
        # create cron dirs if they don't exist
        os.makedirs("/run/cron", exist_ok=True)
        os.makedirs("/tmp/cron", exist_ok=True)

        # prepare crons
        if handler_type == PythonHandlerType and are_models_specified(api_spec):
            cron = FileBasedModelsTreeUpdater(
                interval=10,
                api_spec=api_spec,
                download_dir=model_dir,
            )
            cron.start()
        elif handler_type == TensorFlowHandlerType:
            tf_serving_port = os.getenv("CORTEX_TF_BASE_SERVING_PORT", "9000")
            tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")
            cron = TFSModelLoader(
                interval=10,
                api_spec=api_spec,
                address=f"{tf_serving_host}:{tf_serving_port}",
                tfs_model_dir=model_dir,
                download_dir=model_dir,
            )
            cron.start()
        elif handler_type == TensorFlowNeuronHandlerType:
            cron = prepare_tfs_servers_api(api_spec, model_dir)
            cron.start()

        # wait until the cron finishes its first pass
        if cron:
            while cron.is_alive() and not cron.ran_once():
                time.sleep(0.25)

            # disable live reloading when the BatchAPI kind is used
            # disable live reloading for the TF type when Inferentia is used and when multiple processes are used (num procs > 1)
            if api_spec["kind"] != "RealtimeAPI" or (
                handler_type == TensorFlowNeuronHandlerType
                and has_multiple_tf_servers
                and num_processes > 1
            ):
                cron.stop()

    # to syncronize with the other serving processes
    open("/mnt/workspace/init_script_run.txt", "a").close()

    # don't exit the script if the cron is running
    while cron and cron.is_alive():
        time.sleep(0.25)

    # exit if cron has exited with errors
    if cron and isinstance(cron.exitcode, int) and cron.exitcode != 0:
        # if it was killed by a catchable signal
        if cron.exitcode < 0:
            sys.exit(-cron.exitcode)
        sys.exit(cron.exitcode)


if __name__ == "__main__":
    main()
