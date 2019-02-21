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

CORTEX_VERSION=master

function build_and_upload() {
  set -euo pipefail

  os=$1
  GOOS=$os GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on go build -o cortex github.com/cortexlabs/cortex/cli
  aws s3 cp cortex s3://$CLI_BUCKET_NAME/$CORTEX_VERSION/cli/$os/cortex --only-show-errors
  rm cortex
  echo "Uploaded CLI to s3://$CLI_BUCKET_NAME/$CORTEX_VERSION/cli/$os/cortex"
}

echo ""
echo "Building Cortex CLI for Mac"
build_and_upload darwin

echo ""
echo "Building Cortex CLI for Linux"
build_and_upload linux
