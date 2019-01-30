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
  GOOS=$1
  file=$2

  GOOS=$GOOS GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on go build -installsuffix cgo -o cortex github.com/cortexlabs/cortex/cli

  zip -q $file cortex
  rm cortex

  aws s3 cp $file s3://$CLI_BUCKET_NAME --only-show-errors
  rm $file

  echo "Uploaded $file to s3://$CLI_BUCKET_NAME"
}

###########
### Mac ###
###########

echo ""
echo "Building Cortex CLI for Mac"

build_and_upload darwin "cortex-cli-$CORTEX_VERSION-mac.zip"

#############
### Linux ###
#############

echo ""
echo "Building Cortex CLI for Linux"

build_and_upload linux "cortex-cli-$CORTEX_VERSION-linux.zip"

###############
### Windows ###
###############

echo ""
echo "Building Cortex CLI for Windows"

build_and_upload windows "cortex-cli-$CORTEX_VERSION-windows.zip"
