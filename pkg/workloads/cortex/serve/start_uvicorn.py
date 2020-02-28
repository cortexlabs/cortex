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

if __name__ == "__main__":
    with open("/src/cortex/serve/log_config.yaml", "r") as f:
        log_config = yaml.load(f, yaml.FullLoader)

    # https://github.com/encode/uvicorn/blob/master/uvicorn/config.py
    uvicorn.run(
        "cortex.serve.wsgi:app",
        host="0.0.0.0",
        port=int(os.environ["CORTEX_SERVING_PORT"]),
        workers=int(os.environ["CORTEX_WORKERS_PER_REPLICA"]),
        backlog=int(os.environ["CORTEX_MAX_IN_FLIGHT"]),
        limit_concurrency=int(os.environ["CORTEX_MAX_IN_FLIGHT"]),
        log_config=log_config,
        log_level="info",
    )
