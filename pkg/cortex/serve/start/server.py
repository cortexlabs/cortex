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

import sys
import pathlib
import time

import uvicorn
import yaml


def main():
    uds = sys.argv[1]

    with open("/src/cortex/serve/log_config.yaml", "r") as f:
        log_config = yaml.load(f, yaml.FullLoader)

    while not pathlib.Path("/mnt/workspace/init_script_run.txt").is_file():
        time.sleep(0.2)

    uvicorn.run(
        "cortex_internal.serve.wsgi:app",
        uds=uds,
        forwarded_allow_ips="*",
        proxy_headers=True,
        log_config=log_config,
        log_level="info",
    )


if __name__ == "__main__":
    main()
