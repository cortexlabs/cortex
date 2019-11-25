#!/bin/bash

# Copyright 2019 Cortex Labs, Inc.
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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

CORTEX_VERSION=0.11.0

arg1=${1:-""}
upload="false"
if [ "$arg1" == "upload" ]; then
  upload="true"
fi

function build_and_upload() {
  set -euo pipefail

  os=$1
  echo -e "\nBuilding Cortex CLI for $os"
  GOOS=$os GOARCH=amd64 CGO_ENABLED=0 go build -o cortex "$ROOT/cli"
  if [ "$upload" == "true" ]; then
    echo "Uploading Cortex CLI to s3://$CLI_BUCKET_NAME/$CORTEX_VERSION/cli/$os/cortex"
    aws s3 cp cortex s3://$CLI_BUCKET_NAME/$CORTEX_VERSION/cli/$os/cortex --only-show-errors
  fi
  echo "Done ✓"
  rm cortex
}

build_and_upload darwin

build_and_upload linux
