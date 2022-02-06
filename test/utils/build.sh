#!/bin/bash

# Copyright 2022 Cortex Labs, Inc.
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

# note: this only meant to be called from the build*.sh files in each test api directory
# usage: ./build.sh BUILDER_PATH IMAGE_NAME [REGISTRY] [--skip-push]
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

should_skip_push="false"
positional_args=()
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -s|--skip-push)
    should_skip_push="true"
    shift
    ;;
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done
set -- "${positional_args[@]}"

builder_path="$1"  # e.g. /home/ubuntu/src/github.com/cortexlabs/cortex/test/apis/realtime/hello-world/build-cpu.sh
image_name="$2"  # e.g. realtime-hello-world-cpu
registry=${3:-$CORTEX_DEV_DEFAULT_IMAGE_REGISTRY}  # e.g. 764403040460.dkr.ecr.us-west-2.amazonaws.com/cortexlabs

if [ -z "$registry" ]; then
  error_echo "registry must be provided as a positional arg, or $CORTEX_DEV_DEFAULT_IMAGE_REGISTRY must be set"
fi

image_url="${registry}/${image_name}"  # e.g. 764403040460.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/realtime-sleep-cpu
dockerfile_name="$(echo "$builder_path" | sed 's/.*build-//' | sed 's/\..*//')"  # e.g. cpu
api_dir="$(dirname $builder_path)"  # e.g. /home/ubuntu/src/github.com/cortexlabs/cortex/test/apis/realtime/hello-world
dockerfile_path="${api_dir}/${dockerfile_name}.Dockerfile"  # e.g. /home/ubuntu/src/github.com/cortexlabs/cortex/test/apis/realtime/hello-world/cpu.Dockerfile
if [[ "$registry" == *".ecr."* ]]; then
  login_url="$(echo "$registry" | sed 's/\/.*//')"  # e.g. 764403040460.dkr.ecr.us-west-2.amazonaws.com
  repo_name="$(echo $image_url | sed 's/[^\/]*\///')"  # e.g. cortexlabs/realtime-sleep-cpu
  region="$(echo "$registry" | sed 's/.*\.ecr\.//' | sed 's/\..*//')"  # e.g. us-west-2
fi

if [ ! -f "$dockerfile_path" ]; then
  error_echo "$dockerfile_path does not exist"
  exit 1
fi

blue_echo "Building $image_url:latest\n"
docker build "$api_dir" -f "$dockerfile_path" -t "$image_url"
green_echo "\nBuilt $image_url:latest"

if [ "$should_skip_push" = "true" ]; then
  exit 0
fi

while true; do
  blue_echo "\nPushing $image_url:latest"
  exec 5>&1
  set +e
  out=$(docker push $image_url 2>&1 | tee /dev/fd/5; exit ${PIPESTATUS[0]})
  exit_code=$?
  set -e
  if [ $exit_code -ne 0 ]; then
    if [[ "$image_url" != *".ecr."* ]]; then
      exit $exit_code
    else
      if [[ "$out" == *"authorization token has expired"* ]] || [[ "$out" == *"no basic auth credentials"* ]]; then
        registry_login $login_url $region
        continue
      elif [[ "$out" == *"repository with name"*"does not exist"* ]]; then
        create_ecr_repo $repo_name $region
        continue
      else
        exit $exit_code
      fi
    fi
  fi
  green_echo "\nPushed $image_url:latest"
  break
done

# update api config
find $api_dir -type f -name 'cortex_*.yaml' \
  -exec sed -i "s|quay.io/cortexlabs-test/${image_name}|${image_url}|g" {} \;
