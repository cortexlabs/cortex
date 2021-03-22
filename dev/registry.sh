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

CORTEX_VERSION=0.31.1

set -eo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

source $ROOT/build/images.sh
source $ROOT/dev/util.sh

if [ -f "$ROOT/dev/config/env.sh" ]; then
  source $ROOT/dev/config/env.sh
fi

GCR_HOST=${GCR_HOST:-"gcr.io"}
GCP_PROJECT_ID=${GCP_PROJECT_ID:-}
AWS_ACCOUNT_ID=${AWS_ACCOUNT_ID:-}
AWS_REGION=${AWS_REGION:-}

provider="undefined"
positional_args=()
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -p|--provider)
    provider="$2"
    shift
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
    -p=*|--provider=*)
    provider="${i#*=}"
    shift
    ;;
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
if [ "$provider" = "aws" ]; then
  registry_push_url="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"
elif [ "$provider" = "gcp" ]; then
  registry_push_url="$GCR_HOST/$GCP_PROJECT_ID"
fi

is_registry_logged_in="false"

function registry_login() {
  if [ "$is_registry_logged_in" = "false" ]; then
    if [ "$provider" = "aws" ]; then
      blue_echo "Logging in to ECR..."
      aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $registry_push_url
      aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin 790709498068.dkr.ecr.us-west-2.amazonaws.com  # this is for the inferentia device plugin image
    elif [ "$provider" = "gcp" ]; then
      blue_echo "Logging in to GCS..."
      gcloud auth configure-docker $GCR_HOST
    fi
    is_registry_logged_in="true"
    green_echo "Success\n"
  fi
}

function create_aws_registry() {
  for image in "${aws_images[@]}"; do
    aws ecr create-repository --repository-name=cortexlabs/$image --region=$AWS_REGION || true
  done
}

### HELPERS ###

function build() {
  local image=$1
  local tag=$2
  local dir="${ROOT}/images/${image}"

  build_args=""

  tag_args=""
  if [ -n "$GCP_PROJECT_ID" ]; then
    tag_args+=" -t $GCR_HOST/$GCP_PROJECT_ID/cortexlabs/$image:$tag"
  fi
  if [ -n "$AWS_ACCOUNT_ID" ] && [ -n "$AWS_REGION" ]; then
    tag_args+=" -t $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/cortexlabs/$image:$tag"
  fi

  blue_echo "Building $image:$tag..."
  docker build $ROOT -f $dir/Dockerfile -t cortexlabs/$image:$tag $tag_args $build_args
  green_echo "Built $image:$tag\n"
}

function cache_builder() {
  local image=$1
  local dir="${ROOT}/images/${image}"

  blue_echo "Building $image-builder..."
  docker build $ROOT -f $dir/Dockerfile -t cortexlabs/$image-builder:$CORTEX_VERSION --target builder
  green_echo "Built $image-builder\n"
}

function push() {
  if [ "$provider" == "undefined" ]; then
    return
  fi

  registry_login

  local image=$1
  local tag=$2

  blue_echo "Pushing $image:$tag..."
  docker push $registry_push_url/cortexlabs/$image:$tag
  green_echo "Pushed $image:$tag\n"
}

function build_and_push() {
  local image=$1

  set -euo pipefail  # necessary since this is called in a new shell by parallel

  tag=$CORTEX_VERSION
  if [ "${image}" == "python-predictor-gpu" ]; then
    tag="${CORTEX_VERSION}-cuda10.2-cudnn8"
  fi

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
  local provider=$1

  if [ "$provider" = "aws" ]; then
    if [ -z ${AWS_REGION} ] || [ -z ${AWS_ACCOUNT_ID} ]; then
      echo "error: environment variables AWS_REGION and AWS_ACCOUNT_ID should be exported in dev/config/env.sh"
      exit 1
    fi
  elif [ "$provider" = "gcp" ]; then
    if [ -z ${GCP_PROJECT_ID} ]; then
      echo "error: environment variables GCP_PROJECT_ID should be exported in dev/config/env.sh"
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
validate_env "$provider"

# usage: registry.sh clean --provider aws|gcp
if [ "$cmd" = "clean" ]; then
  if [ "$provider" = "aws" ]; then
    delete_ecr
    create_aws_registry
  fi

# usage: registry.sh create --provider/-p aws|gcp
elif [ "$cmd" = "create" ]; then
  if [ "$provider" = "aws" ]; then
    create_aws_registry
  fi

# usage: registry.sh update-single IMAGE --provider/-p aws|gcp
elif [ "$cmd" = "update-single" ]; then
  image=$sub_cmd
  if [ "$image" = "operator" ] || [ "$image" = "request-monitor" ]; then
    cache_builder $image
  fi
  build_and_push $image

# usage: registry.sh update all|dev|api --provider/-p aws|gcp
# if parallel utility is installed, the docker build commands will be parallelized
elif [ "$cmd" = "update" ]; then
  images_to_build=()

  if [ "$sub_cmd" == "all" ]; then
    images_to_build+=( "${non_dev_images_cluster[@]}" )
    if [ "$provider" == "aws" ]; then
      images_to_build+=( "${non_dev_images_aws[@]}" )
    elif [ "$provider" == "gcp" ]; then
      images_to_build+=( "${non_dev_images_gcp[@]}" )
    elif [ "$provider" == "undefined" ]; then
      images_to_build+=( "${non_dev_images_aws[@]}" "${non_dev_images_gcp[@]}" )
    fi
  fi

  if [[ "$sub_cmd" == "all" || "$sub_cmd" == "dev" ]]; then
    images_to_build+=( "${dev_images_cluster[@]}" )
    if [ "$provider" == "aws" ]; then
      images_to_build+=( "${dev_images_aws[@]}" )
    elif [ "$provider" == "gcp" ]; then
      images_to_build+=( "${dev_images_gcp[@]}" )
    elif [ "$provider" == "undefined" ]; then
      images_to_build+=( "${dev_images_aws[@]}" "${dev_images_aws[@]}" )
    fi
  fi

  images_to_build+=( "${api_images_cluster[@]}" )
  if [ "$provider" == "aws" ]; then
    images_to_build+=( "${api_images_aws[@]}" )
  elif [ "$provider" == "gcp" ]; then
    images_to_build+=( "${api_images_gcp[@]}" )
  elif [ "$provider" == "undefined" ]; then
    images_to_build+=( "${api_images_aws[@]}" "${api_images_gcp[@]}" )
  fi

  if [[ " ${images_to_build[@]} " =~ " operator " ]]; then
    cache_builder operator
  fi
  if [[ " ${images_to_build[@]} " =~ " request-monitor " ]]; then
    cache_builder request-monitor
  fi

  if command -v parallel &> /dev/null && [ -n "${NUM_BUILD_PROCS+set}" ] && [ "$NUM_BUILD_PROCS" != "1" ]; then
    provider=$provider is_registry_logged_in=$is_registry_logged_in ROOT=$ROOT registry_push_url=$registry_push_url SHELL=$(type -p /bin/bash) parallel --will-cite --halt now,fail=1 --eta --jobs $NUM_BUILD_PROCS build_and_push "{}" ::: "${images_to_build[@]}"
  else
    for image in "${images_to_build[@]}"; do
      build_and_push $image
    done
  fi

  cleanup_local

else
  echo "unknown command: $cmd"
  exit 1
fi
