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

set -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

out_file=$ROOT/docs/miscellaneous/cli.md
rm -f $out_file

echo '# CLI commands' >> $out_file
echo '' >> $out_file
echo '_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_' >> $out_file
echo '' >> $out_file
echo '## Install the CLI' >> $out_file
echo '' >> $out_file
echo '```bash' >> $out_file
echo 'pip install cortex' >> $out_file
echo '```' >> $out_file
echo '' >> $out_file
echo '## Install the CLI without Python Client' >> $out_file
echo '' >> $out_file
echo '### Mac/Linux OS' >> $out_file
echo '' >> $out_file
echo '```bash' >> $out_file
echo '# Replace `INSERT_CORTEX_VERSION` with the complete CLI version (e.g. 0.18.1):' >> $out_file
echo '$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/vINSERT_CORTEX_VERSION/get-cli.sh)"' >> $out_file
echo '' >> $out_file
echo '# For example to download CLI version 0.18.1 (Note the "v"):' >> $out_file
echo '$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.18.1/get-cli.sh)"' >> $out_file
echo '```' >> $out_file
echo '' >> $out_file
echo '### Windows' >> $out_file
echo '' >> $out_file
echo 'To install the Cortex CLI on a Windows machine, follow [this guide](../guides/windows-cli.md).' >> $out_file
echo '' >> $out_file

make --no-print-directory -C $ROOT cli

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
  "cluster configure"
  "cluster down"
  "cluster export"
  "env configure"
  "env list"
  "env default"
  "env delete"
  "version"
  "completion"
)

echo '## Command overview' >> $out_file

for cmd in "${commands[@]}"; do
  echo '' >> $out_file
  echo "### ${cmd}" >> $out_file
  echo '' >> $out_file
  echo -n '```text' >> $out_file
  $ROOT/bin/cortex help ${cmd} >> $out_file
  echo '```' >> $out_file
done

# Bring back CLI config
mv -f $cli_config_backup_path $HOME/.cortex/cli.yaml

echo "updated $out_file"
