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

# usage: ./build.sh build PATH [REGISTRY] [--skip-push]
#   PATH is e.g. /home/ubuntu/src/github.com/cortexlabs/cortex/test/apis/realtime/sleep/build-cpu.sh
#   REGISTRY defaults to $CORTEX_DEV_DEFAULT_IMAGE_REGISTRY; e.g. 764403040460.dkr.ecr.us-west-2.amazonaws.com/cortexlabs or quay.io/cortexlabs-test

set -eo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. >/dev/null && pwd)"
source $ROOT/dev/util.sh

function registry_login() {
  login_url=$1  # e.g. 764403040460.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/realtime-sleep-cpu
  region=$2

  blue_echo "\nLogging in to ECR"
  aws ecr get-login-password --region $region | docker login --username AWS --password-stdin $login_url
  green_echo "\nSuccess"
}

function create_ecr_repo() {
  repo_name=$1  # e.g. cortexlabs/realtime-sleep-cpu
  region=$2

  blue_echo "\nCreating ECR repo $repo_name"
  aws ecr create-repository --repository-name=$repo_name --region=$region
  green_echo "\nSuccess"
}

path="$1"
registry="$CORTEX_DEV_DEFAULT_IMAGE_REGISTRY"
should_skip_push="false"
for arg in "${@:2}"; do
  if [ "$arg" = "--skip-push" ]; then
    should_skip_push="true"
  else
    registry="$arg"
  fi
done

if [ -z "$registry" ]; then
  error_echo "registry must be provided as a positional arg, or $CORTEX_DEV_DEFAULT_IMAGE_REGISTRY must be set"
fi

name="$(basename $(dirname "$path"))"  # e.g. sleep
kind="$(basename $(dirname $(dirname "$path")))"  # e.g. realtime
architecture="$(echo "$(basename "$path" .sh)" | sed 's/.*-//')"  # e.g. cpu
image_url="$registry/$kind-$name-$architecture"  # e.g. 764403040460.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/realtime-sleep-cpu
if [[ "$image_url" == *".ecr."* ]]; then
  login_url="$(echo "$image_url" | sed 's/\/.*//')"  # e.g. 764403040460.dkr.ecr.us-west-2.amazonaws.com
  repo_name="$(echo $image_url | sed 's/[^\/]*\///')"  # e.g. cortexlabs/realtime-sleep-cpu
  region="$(echo "$image_url" | sed 's/.*\.ecr\.//' | sed 's/\..*//')"  # e.g. us-west-2
fi

blue_echo "Building $image_url:latest\n"
docker build "$(dirname "$path")" -f "$(dirname "$path")/$architecture.Dockerfile" -t "$image_url"
green_echo "\nBuilt $image_url:latest"

if [ "$should_skip_push" = "true" ]; then
  exit 0
fi

while true; do
  blue_echo "\nPushing $image_url:latest"
  exec 5>&1
  set +e
  out=$(docker push $image_url 2>&1 | tee /dev/fd/5; exit ${PIPESTATUS[0]})
  set -e
  if [[ "$image_url" == *".ecr."* ]]; then
    if [[ "$out" == *"authorization token has expired"* ]] || [[ "$out" == *"no basic auth credentials"* ]]; then
      registry_login $login_url $region
      continue
    elif [[ "$out" == *"does not exist"* ]]; then
      create_ecr_repo $repo_name $region
      continue
    fi
  fi
  green_echo "\nPushed $image_url:latest"
  break
done
