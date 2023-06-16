# Copyright 2022 Cortex Labs, Inc.
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

# Usage: python create_user.py $KUBE_PROXY_CONFIG.yaml

import yaml
import sys


def main():
    kube_proxy_config_file = sys.argv[1]
    with open(kube_proxy_config_file, "r") as f:
        kube_proxy_config = yaml.safe_load(f)
    kube_proxy_config = yaml.safe_load(kube_proxy_config["data"]["config"])

    kube_proxy_config["mode"] = "ipvs"  # IP Virtual Server
    kube_proxy_config["ipvs"]["scheduler"] = "rr"  # round robin

    print(yaml.dump(kube_proxy_config, indent=2))


if __name__ == "__main__":
    main()
