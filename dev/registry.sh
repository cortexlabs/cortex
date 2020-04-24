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

source $ROOT/dev/config/build.sh
source $ROOT/dev/util.sh

ecr_logged_in=false

function ecr_login() {
  if [ "$ecr_logged_in" = false ]; then
    blue_echo "Logging in to ECR..."
    ecr_login_command=$(aws ecr get-login --no-include-email --region $REGISTRY_REGION)
    eval $ecr_login_command
    ecr_logged_in=true
    green_echo "Success\n"
  fi
}

function create_registry() {
  aws ecr create-repository --repository-name=cortexlabs/python-serve --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/python-serve-gpu --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/tf-serve --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/tf-serve-gpu --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/tf-api --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/onnx-serve --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/onnx-serve-gpu --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/operator --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/manager --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/downloader --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/cluster-autoscaler --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/metrics-server --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/nvidia --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/fluentd --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/statsd --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/istio-proxy --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/istio-pilot --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/istio-citadel --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/istio-galley --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/request-monitor --region=$REGISTRY_REGION || true
}

### HELPERS ###

function build() {
  dir=$1
  image=$2
  tag=$3

  blue_echo "Building $image:$tag..."
  docker build $ROOT -f $dir/Dockerfile -t cortexlabs/$image:$tag -t $REGISTRY_URL/cortexlabs/$image:$tag
  green_echo "Built $image:$tag\n"
}

function build_base() {
  dir=$1
  image=$2

  blue_echo "Building $image..."
  docker build $ROOT -f $dir/Dockerfile -t cortexlabs/$image:latest
  green_echo "Built $image\n"
}

function cache_builder() {
  dir=$1
  image=$2

  blue_echo "Building $image-builder..."
  docker build $ROOT -f $dir/Dockerfile -t cortexlabs/$image-builder:latest --target builder
  green_echo "Built $image-builder\n"
}

function push() {
  ecr_login

  image=$1
  tag=$2

  blue_echo "Pushing $image:$tag..."
  docker push $REGISTRY_URL/cortexlabs/$image:$tag
  green_echo "Pushed $image:$tag\n"
}

function build_and_push() {
  dir=$1
  image=$2
  tag=$3

  build $dir $image $tag
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

cmd=${1:-""}
env=${2:-""}

if [ "$cmd" = "clean" ]; then
  cleanup_local
  cleanup_ecr

elif [ "$cmd" = "create" ]; then
  create_registry

elif [ "$cmd" = "update-manager-local" ]; then
  build $ROOT/images/manager manager latest

elif [ "$cmd" = "update" ]; then
  if [ "$env" != "dev" ]; then
    build_and_push $ROOT/images/tf-serve tf-serve latest
    build_and_push $ROOT/images/tf-serve-gpu tf-serve-gpu latest

    cache_builder $ROOT/images/operator operator
    build_and_push $ROOT/images/operator operator latest

    build_and_push $ROOT/images/cluster-autoscaler cluster-autoscaler latest
    build_and_push $ROOT/images/metrics-server metrics-server latest
    build_and_push $ROOT/images/nvidia nvidia latest
    build_and_push $ROOT/images/fluentd fluentd latest
    build_and_push $ROOT/images/statsd statsd latest
    build_and_push $ROOT/images/istio-proxy istio-proxy latest
    build_and_push $ROOT/images/istio-pilot istio-pilot latest
    build_and_push $ROOT/images/istio-citadel istio-citadel latest
    build_and_push $ROOT/images/istio-galley istio-galley latest
  fi

  cache_builder $ROOT/images/request-monitor request-monitor
  build_and_push $ROOT/images/request-monitor request-monitor latest

  build_and_push $ROOT/images/manager manager latest
  build_and_push $ROOT/images/python-serve python-serve latest
  build_and_push $ROOT/images/python-serve-gpu python-serve-gpu latest
  build_and_push $ROOT/images/tf-api tf-api latest
  build_and_push $ROOT/images/onnx-serve onnx-serve latest
  build_and_push $ROOT/images/onnx-serve-gpu onnx-serve-gpu latest
  build_and_push $ROOT/images/downloader downloader latest

  cleanup_local
fi
