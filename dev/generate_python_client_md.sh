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


set -euo pipefail

if [[ "$OSTYPE" != "linux"* ]]; then
  echo "error: this script is only designed to run on linux"
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"
docs_path="$ROOT/docs/clients/python.md"

pip3 uninstall -y cortex

cd $ROOT/pkg/cortex/client

pip3 install -e .

pydoc-markdown -m cortex -m cortex.client --render-toc > $docs_path

# title
sed -i "s/# Table of Contents/# Python API/g" $docs_path

# delete links
sed -i "/<a name=/d" $docs_path

# delete unnecessary section headers
sed -i "/_](#cortex\.client\.Client\.__init__)/d" $docs_path
sed -i "/\* \[Client](#cortex\.client\.Client)/d" $docs_path
# fix section link/header
sed -i "s/\* \[cortex\.client](#cortex\.client)/\* [cortex\.client\.Client](#cortex-client-client)/g" $docs_path
sed -i "s/# cortex\.client/# cortex\.client\.Client/g" $docs_path
# delete unnecessary section body
sed -i "/# cortex.client.Client/,/## create\\\_api/{//!d}" $docs_path
sed -i "s/# cortex.client.Client/# cortex.client.Client\n/g" $docs_path

# fix table of contents links
sed -i "s/](#cortex\./](#/g" $docs_path
sed -i "s/](#client\.Client\./](#/g" $docs_path

# indentation
sed -i "s/    \* /  \* /g" $docs_path
sed -i "s/#### /## /g" $docs_path

# whitespace
sed -i 's/[[:space:]]*$//' $docs_path
truncate -s -1 $docs_path

# Cortex version comment
sed -i "s/^## create\\\_api/## create\\\_api\n\n<!-- CORTEX_VERSION_MINOR -->/g" $docs_path

pip3 uninstall -y cortex
rm -rf $ROOT/pkg/cortex/client/cortex.egg-info
