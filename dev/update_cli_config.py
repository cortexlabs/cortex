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
import os


def update_cli_config(cli_config_file_path, env_name, provider, operator_endpoint):
    if provider == "aws":
        new_env = {
            "name": env_name,
            "provider": provider,
            "operator_endpoint": operator_endpoint,
            "aws_access_key_id": os.getenv("AWS_ACCESS_KEY_ID"),
            "aws_secret_access_key": os.getenv("AWS_SECRET_ACCESS_KEY"),
        }
    else:
        new_env = {
            "name": env_name,
            "provider": provider,
            "operator_endpoint": operator_endpoint,
        }

    try:
        with open(cli_config_file_path, "r") as f:
            cli_config = yaml.safe_load(f)
            if cli_config is None:
                raise Exception("blank cli config file")
    except:
        cli_config = {"environments": [new_env]}
        with open(cli_config_file_path, "w") as f:
            yaml.dump(cli_config, f, default_flow_style=False)
        return

    if len(cli_config.get("environments", [])) == 0:
        cli_config["environments"] = [new_env]
        with open(cli_config_file_path, "w") as f:
            yaml.dump(cli_config, f, default_flow_style=False)
        return

    replaced = False
    for i, prev_env in enumerate(cli_config["environments"]):
        if prev_env.get("name") == env_name:
            cli_config["environments"][i] = new_env
            replaced = True
            break

    if not replaced:
        cli_config["environments"].append(new_env)

    with open(cli_config_file_path, "w") as f:
        yaml.dump(cli_config, f, default_flow_style=False)


if __name__ == "__main__":
    update_cli_config(
        cli_config_file_path=sys.argv[1],
        env_name=sys.argv[2],
        provider=sys.argv[3],
        operator_endpoint=sys.argv[4],
    )
