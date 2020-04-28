#!/bin/bash

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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

# Clear default environments
cli_config_backup_path=$HOME/.cortex/cli-bak-$RANDOM.yaml
mv $HOME/.cortex/cli.yaml $cli_config_backup_path

commands=(
  "deploy"
  "get"
  "logs"
  "refresh"
  "predict"
  "delete"
  "cluster up"
  "cluster info"
  "cluster update"
  "cluster down"
  "env configure"
  "env list"
  "env default"
  "env delete"
  "version"
  "completion"
)

for cmd in "${commands[@]}"; do
  echo "## ${cmd}"
  echo
  echo -n '```text'
  $ROOT/bin/cortex help ${cmd}
  echo '```'
  echo
done

# Bring back CLI config
mv -f $cli_config_backup_path $HOME/.cortex/cli.yaml
