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

import os
import time
import json

from cortex.lib.type import (
    predictor_type_from_api_spec,
    PythonPredictorType,
    TensorFlowPredictorType,
    TensorFlowNeuronPredictorType,
    ONNXPredictorType,
)
from cortex.lib.model import (
    FileBasedModelsTreeUpdater,  # only when num workers > 1
    TFSModelLoader,
)
from cortex.lib.api import get_spec
from cortex.lib.checkers.pod import wait_neuron_rtd


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
    if api_spec["predictor"]["model_path"] is not None:
        return True

    if api_spec["predictor"]["models"] and (
        api_spec["predictor"]["models"]["dir"] is not None
        or len(api_spec["predictor"]["models"]["paths"]) > 0
    ):
        return True
    return False


def is_model_caching_enabled(api_spec: dir) -> bool:
    return (
        api_spec["predictor"]["models"]
        and api_spec["predictor"]["models"]["cache_size"] is not None
        and api_spec["predictor"]["models"]["disk_cache_size"] is not None
    )


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
    provider = os.environ["CORTEX_PROVIDER"]
    spec_path = os.environ["CORTEX_API_SPEC"]
    cache_dir = os.getenv("CORTEX_CACHE_DIR")  # when it's deployed locally
    bucket = os.getenv("CORTEX_BUCKET")  # when it's deployed to AWS
    region = os.getenv("AWS_REGION")  # when it's deployed to AWS
    _, api_spec = get_spec(provider, spec_path, cache_dir, bucket, region)

    predictor_type = predictor_type_from_api_spec(api_spec)
    multiple_processes = api_spec["predictor"]["processes_per_replica"] > 1
    caching_enabled = is_model_caching_enabled(api_spec)
    model_dir = os.getenv("CORTEX_MODEL_DIR")

    # start live-reloading when model caching not enabled > 1
    cron = None
    if not caching_enabled:
        # create cron dirs if they don't exist
        os.makedirs("/run/cron", exist_ok=True)
        os.makedirs("/tmp/cron", exist_ok=True)

        # prepare crons
        if predictor_type in [PythonPredictorType, ONNXPredictorType] and are_models_specified(
            api_spec
        ):
            cron = FileBasedModelsTreeUpdater(
                interval=10,
                api_spec=api_spec,
                download_dir=model_dir,
            )
            cron.start()
        elif predictor_type == TensorFlowPredictorType:
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
        elif predictor_type == TensorFlowNeuronPredictorType:
            cron = prepare_tfs_servers_api(api_spec, model_dir)
            cron.start()

        # wait until the cron finishes its first pass
        if cron:
            while not cron.ran_once():
                time.sleep(0.25)

            # disable live reloading when the BatchAPI kind is used
            # disable live reloading for the TF predictor when Inferentia is used and when multiple processes are used
            if api_spec["kind"] != "RealtimeAPI" or (
                predictor_type == TensorFlowNeuronPredictorType and has_multiple_tf_servers
            ):
                cron.stop()


if __name__ == "__main__":
    main()
