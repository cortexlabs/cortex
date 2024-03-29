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

CORTEX_VERSION=master

set -eo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

source $ROOT/build/images.sh
source $ROOT/dev/util.sh

images_that_can_run_locally="operator manager"

if [ -f "$ROOT/dev/config/env.sh" ]; then
  source $ROOT/dev/config/env.sh
fi

AWS_ACCOUNT_ID=${AWS_ACCOUNT_ID:-}
AWS_REGION=${AWS_REGION:-}

skip_push="false"
include_arm64_arch="false"
positional_args=()
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --skip-push)
    skip_push="true"
    shift
    ;;
    --include-arm64-arch)
    include_arm64_arch="true"
    shift
    ;;
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done
set -- "${positional_args[@]}"
positional_args=()
for i in "$@"; do
  case $i in
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done
set -- "${positional_args[@]}"
for arg in "$@"; do
  if [[ "$arg" == -* ]]; then
    echo "unknown flag: $arg"
    exit 1
  fi
done

cmd=${1:-""}
sub_cmd=${2:-""}

registry_push_url=""
if [ "$skip_push" != "true" ]; then
  registry_push_url="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"
fi

is_registry_logged_in="false"

function registry_login() {
  if [ "$is_registry_logged_in" = "false" ]; then
    blue_echo "Logging in to ECR..."
    aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $registry_push_url
    is_registry_logged_in="true"
    green_echo "Success\n"
  fi
}

function create_ecr_repository() {
  for image in "${all_images[@]}"; do
    aws ecr create-repository --repository-name=cortexlabs/$image --region=$AWS_REGION || true
  done
}

### HELPERS ###

function build_and_push() {
  local image=$1
  local include_arm64_arch=$2
  local dir="${ROOT}/images/${image}"

  set -euo pipefail

  if [[ ! " ${multi_arch_images[*]} " =~ " $image " ]]; then
    include_arm64_arch="false"
  fi

  if [ ! -n "$AWS_ACCOUNT_ID" ] || [ ! -n "$AWS_REGION" ]; then
    echo "AWS_ACCOUNT_ID or AWS_REGION env vars not found"
    exit 1
  fi

  tag=$CORTEX_VERSION

  push_or_not_flag=""
  running_operation="Building"
  finished_operation="Built"
  if [ "$skip_push" = "false" ]; then
    push_or_not_flag="--push"
    running_operation+=" and pushing"
    finished_operation+=" and pushed"
    registry_login
  fi

  if [ "$include_arm64_arch" = "true" ]; then
    blue_echo "$running_operation $image:$tag (amd64 and arm64)..."
  else
    blue_echo "$running_operation $image:$tag (amd64)..."
  fi

  platforms="linux/amd64"
  if [ "$include_arm64_arch" = "true" ]; then
    platforms+=",linux/arm64"
  fi

  docker buildx build $ROOT -f $dir/Dockerfile -t $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/cortexlabs/$image:$tag --platform $platforms $push_or_not_flag

  if [ "$include_arm64_arch" = "true" ]; then
    green_echo "$finished_operation $image:$tag (amd64 and arm64)"
  else
    green_echo "$finished_operation $image:$tag (amd64)"
  fi

  if [[ " ${images_that_can_run_locally[*]} " =~ " $image " ]] && [[ "$include_arm64_arch" == "false" ]]; then
    blue_echo "Exporting $image:$tag to local docker..."
    docker buildx build $ROOT -f $dir/Dockerfile -t cortexlabs/$image:$tag -t $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/cortexlabs/$image:$tag --platform $platforms --load
    green_echo "Exported $image:$tag to local docker"
  fi
}

function cleanup_local() {
  echo "cleaning local repositories..."
  docker container prune -f
  docker image prune -f
  docker buildx prune -f
}

function cleanup_ecr() {
  echo "cleaning ECR repositories..."
  repos=$(aws ecr describe-repositories --output text | awk '{print $6}' | grep -P "\S")
  echo "$repos" |
  while IFS= read -r repo; do
    imageIDs=$(aws ecr list-images --repository-name "$repo" --filter tagStatus=UNTAGGED --query "imageIds[*]" --output text)
    echo "$imageIDs" |
    while IFS= read -r imageId; do
      if [ ! -z "$imageId" ]; then
        echo "Removing from ECR: $repo/$imageId"
        aws ecr batch-delete-image --repository-name "$repo" --image-ids imageDigest="$imageId" >/dev/null;
      fi
    done
  done
}

function delete_ecr() {
  echo "deleting ECR repositories..."
  repos=$(aws ecr describe-repositories --output text | awk '{print $6}' | grep -P "\S")
  echo "$repos" |
  while IFS= read -r repo; do
    imageIDs=$(aws ecr delete-repository --force --repository-name "$repo")
    echo "deleted: $repo"
  done
}

function validate_env() {
  if [ "$skip_push" != "true" ]; then
    if [ -z ${AWS_REGION} ] || [ -z ${AWS_ACCOUNT_ID} ]; then
      echo "error: environment variables AWS_REGION and AWS_ACCOUNT_ID should be exported in dev/config/env.sh"
      exit 1
    fi
  fi
}

# validate environment is correctly set on env.sh
validate_env

# usage: registry.sh clean
if [ "$cmd" = "clean" ]; then
  delete_ecr
  create_ecr_repository

# usage: registry.sh create
elif [ "$cmd" = "create" ]; then
  create_ecr_repository

# usage: registry.sh update-single IMAGE
elif [ "$cmd" = "update-single" ]; then
  image=$sub_cmd
  build_and_push $image $include_arm64_arch

# usage: registry.sh update all|dev|api
elif [ "$cmd" = "update" ]; then
  images_to_build=()

  if [ "$sub_cmd" == "all" ]; then
    images_to_build+=( "${non_dev_images[@]}" )
  fi

  if [[ "$sub_cmd" == "all" || "$sub_cmd" == "dev" ]]; then
    images_to_build+=( "${dev_images[@]}" )
  fi

  for image in "${images_to_build[@]}"; do
    build_and_push $image $include_arm64_arch
  done

# usage: registry.sh clean-cache
elif [ "$cmd" = "clean-cache" ]; then
  cleanup_local

else
  echo "unknown command: $cmd"
  exit 1
fi
