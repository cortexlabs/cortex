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
import yaml


def remove_cli_config(cli_config_file_path, operator_endpoint):
    removed_env_names = []

    with open(cli_config_file_path, "r") as f:
        cli_config = yaml.safe_load(f)

    if cli_config is None:
        return

    prev_envs = cli_config.get("environments", [])
    updated_envs = []

    for prev_env in prev_envs:
        if prev_env.get("operator_endpoint", "").endswith(operator_endpoint):
            removed_env_names.append(prev_env["name"])
        else:
            updated_envs.append(prev_env)

    if len(updated_envs) == len(prev_envs):
        return

    cli_config["environments"] = updated_envs

    prev_default = cli_config.get("default_environment")
    if prev_default in removed_env_names:
        cli_config["default_environment"] = "local"

    with open(cli_config_file_path, "w") as f:
        yaml.dump(cli_config, f, default_flow_style=False)

    if len(removed_env_names) == 1:
        print(f"✓ deleted the {removed_env_names[0]} environment configuration")
    elif len(removed_env_names) == 2:
        print(
            f"✓ deleted the {removed_env_names[0]} and {removed_env_names[1]} environment configurations"
        )
    elif len(removed_env_names) > 2:
        print(
            f"✓ deleted the {', '.join(removed_env_names[:-1])}, and {removed_env_names[-1]} environment configurations"
        )

    if prev_default in removed_env_names:
        print(f"✓ set the default environment to local")


# this is best effort, will not error in any circumstances
if __name__ == "__main__":
    try:
        remove_cli_config(cli_config_file_path=sys.argv[1], operator_endpoint=sys.argv[2])
    except:
        pass
