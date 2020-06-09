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

import uvicorn
import yaml
import os

from cortex.lib.type import get_spec
from cortex.lib.storage import S3, LocalStorage


def load_tensorflow_serving_models():
    # get TFS address
    tf_serving_port = os.getenv("CORTEX_TF_SERVING_PORT", "9000")
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")
    tf_serving_address = tf_serving_host + ":" + tf_serving_port

    # get models from environment variable
    models = os.environ["CORTEX_MODELS"].split(",")
    models = [model.strip() for model in models]

    from cortex.lib.server.tensorflow import TensorFlowServing

    # load models
    tfs = TensorFlowServing(tf_serving_address)
    model_dir = os.environ["CORTEX_MODEL_DIR"]
    base_paths = [os.path.join(model_dir, name) for name in models]
    tfs.add_models_config(models, base_paths, replace_models=False)


def main():
    with open("/src/cortex/serve/log_config.yaml", "r") as f:
        log_config = yaml.load(f, yaml.FullLoader)

    # get API spec
    cache_dir = os.environ["CORTEX_CACHE_DIR"]
    provider = os.environ["CORTEX_PROVIDER"]
    spec_path = os.environ["CORTEX_API_SPEC"]
    if provider == "local":
        storage = LocalStorage(os.getenv("CORTEX_CACHE_DIR"))
    else:
        storage = S3(bucket=os.environ["CORTEX_BUCKET"], region=os.environ["AWS_REGION"])
    raw_api_spec = get_spec(provider, storage, cache_dir, spec_path)

    # load tensorflow models into TFS
    if raw_api_spec["predictor"]["type"] == "tensorflow":
        load_tensorflow_serving_models()

    # https://github.com/encode/uvicorn/blob/master/uvicorn/config.py
    uvicorn.run(
        "cortex.serve.wsgi:app",
        host="0.0.0.0",
        port=int(os.environ["CORTEX_SERVING_PORT"]),
        workers=int(os.environ["CORTEX_WORKERS_PER_REPLICA"]),
        limit_concurrency=int(os.environ["CORTEX_MAX_WORKER_CONCURRENCY"]),
        backlog=int(os.environ["CORTEX_SO_MAX_CONN"]),
        log_config=log_config,
        log_level="info",
    )


if __name__ == "__main__":
    main()
