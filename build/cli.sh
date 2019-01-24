#!/bin/bash
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
