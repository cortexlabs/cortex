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


set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

source $ROOT/build/images.sh
source $ROOT/dev/config/build.sh
source $ROOT/dev/util.sh

flag_include_slim="false"
flag_skip_push="false"
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --include-slim)
    flag_include_slim="true"
    shift
    ;;
    --skip-push)
    flag_skip_push="true"
    shift
    ;;
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done
set -- "${positional_args[@]}"

cmd=${1:-""}
sub_cmd=${2:-""}

ecr_logged_in=false

function ecr_login() {
  if [ "$ecr_logged_in" = false ]; then
    blue_echo "Logging in to ECR..."
    aws ecr get-login-password --region $REGISTRY_REGION | docker login --username AWS --password-stdin $REGISTRY_URL
    aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin 790709498068.dkr.ecr.us-west-2.amazonaws.com  # this is for the inferentia device plugin image
    ecr_logged_in=true
    green_echo "Success\n"
  fi
}

function create_registry() {
  for image in "${all_images[@]}"; do
    aws ecr create-repository --repository-name=cortexlabs/$image --region=$REGISTRY_REGION || true
  done
}

### HELPERS ###

function build() {
  local image=$1
  local tag=$2
  local dir="${ROOT}/images/${image/-slim}"

  build_args=""
  if [[ "$image" == *"-slim" ]]; then
    build_args="--build-arg SLIM=true"
  fi

  blue_echo "Building $image:$tag..."
  docker build $ROOT -f $dir/Dockerfile -t cortexlabs/$image:$tag -t $REGISTRY_URL/cortexlabs/$image:$tag $build_args
  green_echo "Built $image:$tag\n"
}

function cache_builder() {
  local image=$1
  local dir="${ROOT}/images/${image/-slim}"

  blue_echo "Building $image-builder..."
  docker build $ROOT -f $dir/Dockerfile -t cortexlabs/$image-builder:latest --target builder
  green_echo "Built $image-builder\n"
}

function push() {
  if [ "$flag_skip_push" == "true" ]; then
    return
  fi

  ecr_login

  local image=$1
  local tag=$2

  blue_echo "Pushing $image:$tag..."
  docker push $REGISTRY_URL/cortexlabs/$image:$tag
  green_echo "Pushed $image:$tag\n"
}

function build_and_push() {
  local image=$1
  local tag=$2

  set -euo pipefail  # necessary since this is called in a new shell by parallel

  build $image $tag
  push $image $tag
}

function cleanup_local() {
  echo "cleaning local repositories..."
  docker container prune -f
  docker image prune -f
}

function cleanup_ecr() {
  echo "cleaning ECR repositories..."
  repos=$(aws ecr describe-repositories --output text | awk '{print $6}' | grep -P "\S")
  echo "$repos" |
  while IFS= read -r line; do
    imageIDs=$(aws ecr list-images --repository-name "$line" --filter tagStatus=UNTAGGED --query "imageIds[*]" --output text)
    echo "$imageIDs" |
    while IFS= read -r imageId; do
      if [ ! -z "$imageId" ]; then
        echo "Removing from ECR: $line/$imageId"
        aws ecr batch-delete-image --repository-name "$line" --image-ids imageDigest="$imageId" >/dev/null;
      fi
    done
  done
}

# export functions for parallel command
export -f build_and_push
export -f push
export -f build
export -f blue_echo
export -f green_echo
export -f ecr_login

if [ "$cmd" = "clean" ]; then
  cleanup_local
  cleanup_ecr

elif [ "$cmd" = "create" ]; then
  create_registry

elif [ "$cmd" = "update-manager-local" ]; then
  build manager latest

# usage: registry.sh update all|dev|api [--include-slim] [--skip-push]
# if parallel utility is installed, the docker build commands will be parallelized
elif [ "$cmd" = "update" ]; then
  images_to_build=()

  if [ "$sub_cmd" == "all" ]; then
    cache_builder operator
    images_to_build+=( "${non_dev_images[@]}" )
  fi

  if [[ "$sub_cmd" == "all" || "$sub_cmd" == "dev" ]]; then
    cache_builder request-monitor
    images_to_build+=( "${dev_images[@]}" )
  fi

  images_to_build+=( "${user_facing_images[@]}" )

  if [ "$flag_include_slim" == "true" ]; then
    images_to_build+=( "${user_facing_slim_images[@]}" )
  fi

  if command -v parallel &> /dev/null ; then
    flag_skip_push=$flag_skip_push ecr_logged_in=$ecr_logged_in ROOT=$ROOT REGISTRY_URL=$REGISTRY_URL SHELL=$(type -p /bin/bash) parallel --will-cite --halt now,fail=1 --eta build_and_push "{} latest" ::: "${images_to_build[@]}"
  else
    for image in "${images_to_build[@]}"; do
      build_and_push $image latest
    done
  fi

  cleanup_local
fi
