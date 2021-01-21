#!/bin/bash

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

set -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

out_file=$ROOT/docs/clients/cli.md
rm -f $out_file

echo "building cli ..."
make --no-print-directory -C $ROOT cli

# Clear default environments
cli_config_backup_path=$HOME/.cortex/cli-bak-$RANDOM.yaml
mv $HOME/.cortex/cli.yaml $cli_config_backup_path

echo "# CLI commands" >> $out_file

commands=(
  "deploy"
  "get"
  "logs"
  "patch"
  "refresh"
  "delete"
  "cluster up"
  "cluster info"
  "cluster configure"
  "cluster down"
  "cluster export"
  "cluster-gcp up"
  "cluster-gcp info"
  "cluster-gcp down"
  "env configure"
  "env list"
  "env default"
  "env delete"
  "version"
  "completion"
)

echo "running help commands ..."

for cmd in "${commands[@]}"; do
  echo '' >> $out_file
  echo "## ${cmd}" >> $out_file
  echo '' >> $out_file
  echo '```text' >> $out_file
  $ROOT/bin/cortex help ${cmd} >> $out_file
  echo '```' >> $out_file
done

# Bring back CLI config
mv -f $cli_config_backup_path $HOME/.cortex/cli.yaml

echo "updated $out_file"
