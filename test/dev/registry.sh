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

set -eo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. >/dev/null && pwd)"

source $ROOT/dev/util.sh
source $ROOT/test/dev/images.sh

if [ -f "$ROOT/dev/config/env.sh" ]; then
  source $ROOT/dev/config/env.sh
fi

AWS_ACCOUNT_ID=${AWS_ACCOUNT_ID:-}
AWS_REGION=${AWS_REGION:-}

skip_push="false"
positional_args=()
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --skip-push)
    skip_push="true"
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
  for image in "${api_images[@]}"; do
    aws ecr create-repository --repository-name=cortexlabs-tests/$image --region=$AWS_REGION || true
  done
}

### HELPERS ###

function build() {
  local image=$1
  local tag=$2

  tag_args=""
  if [ -n "$AWS_ACCOUNT_ID" ] && [ -n "$AWS_REGION" ]; then
    tag_args+=" -t $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/cortexlabs-tests/$image:$tag"
  fi

  local num_words="$(awk -F'-' '{ for(i=1;i<=NF;i++) print $i }' <<< ${image} | wc -l | sed -e 's/^[[:space:]]*//')"
  local kind_dir="$( cut -d '-' -f 1 <<< "${image}" )"
  local api_name="$( cut -d '-' -f 2-$((num_words-1)) <<< "${image}" )"
  local compute_type="$( cut -d '-' -f ${num_words} <<< "${image}" )"

  local dir="${ROOT}/test/apis/${kind_dir}/${api_name}"
  echo "docker build $dir -f $api_name-$compute_type.dockerfile -t cortexlabs-tests/$image:$tag $tag_args"
  docker build $dir -f $dir/$api_name-$compute_type.dockerfile -t cortexlabs-tests/$image:$tag $tag_args
  green_echo "Built $image:$tag\n"
}

function push() {
  if [ "$skip_push" = "true" ]; then
    return
  fi

  registry_login

  local image=$1
  local tag=$2

  blue_echo "Pushing $image:$tag..."
  docker push $registry_push_url/cortexlabs-tests/$image:$tag
  green_echo "Pushed $image:$tag\n"
}

function build_and_push() {
  local image=$1

  set -euo pipefail  # necessary since this is called in a new shell by parallel

  tag=latest
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

# export functions for parallel command
export -f build_and_push
export -f push
export -f build
export -f blue_echo
export -f green_echo
export -f registry_login

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
  build_and_push $image

# usage: registry.sh update
# if parallel utility is installed, the docker build commands will be parallelized
elif [ "$cmd" = "update" ]; then
  # build api images
  if command -v parallel &> /dev/null && [ -n "${NUM_BUILD_PROCS+set}" ] && [ "$NUM_BUILD_PROCS" != "1" ]; then
    is_registry_logged_in=$is_registry_logged_in ROOT=$ROOT registry_push_url=$registry_push_url SHELL=$(type -p /bin/bash) parallel --will-cite --halt now,fail=1 --eta --jobs $NUM_BUILD_PROCS build_and_push "{}" ::: "${api_images[@]}"
  else
    for image in "${api_images[@]}"; do
      build_and_push $image
    done
  fi
  cleanup_local

else
  echo "unknown command: $cmd"
  exit 1
fi
