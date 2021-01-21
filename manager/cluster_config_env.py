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

import sys
import yaml
import json
from copy import deepcopy


def export(base_key, value):
    if base_key.lower() == "cortex_tags":
        exportTags(value, "CORTEX_TAGS")
        exportTags(
            value, "CORTEX_OPERATOR_LOAD_BALANCER_TAGS", {"cortex.dev/load-balancer": "operator"}
        )
        exportTags(value, "CORTEX_API_LOAD_BALANCER_TAGS", {"cortex.dev/load-balancer": "api"})
        return

    if value is None:
        return
    elif type(value) is list:
        print(f'export {base_key.upper()}="{yaml.dump(value, default_flow_style=True).strip()}"')
    elif type(value) is dict:
        for key, child in value.items():
            export(base_key + "_" + key, child)
    else:
        print(f'export {base_key.upper()}="{value}"')
        if base_key.lower().startswith("cortex_image_"):
            hub = value.rsplit("/", 1)[0]
            suffix = value.rsplit("/", 1)[1]
            if ":" in suffix:
                image = suffix.split(":")[0]
                tag = suffix.split(":")[1]
            else:
                image = suffix
                tag = "latest"

            print(f'export {base_key.upper()}_HUB="{hub}"')
            print(f'export {base_key.upper()}_IMAGE="{image}"')
            print(f'export {base_key.upper()}_TAG="{tag}"')


def exportTags(tags, env_var_name, tag_overrides={}):
    tags = deepcopy(tags)
    for k, v in tag_overrides.items():
        tags[k] = v
    inlined_tags = ",".join([f"{k}={v}" for k, v in tags.items()])
    print(f"export {env_var_name}='{inlined_tags}'")
    print(f"export {env_var_name}_JSON='{json.dumps(tags)}'")


for config_path in sys.argv[1:]:
    with open(config_path, "r") as f:
        config = yaml.safe_load(f)

    export("CORTEX", config)
