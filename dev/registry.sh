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
  aws ecr create-repository --repository-name=cortexlabs/manager --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/argo-controller --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/argo-executor --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/fluentd --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/nginx-backend --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/nginx-controller --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/operator --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/spark --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/spark-operator --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/tf-serve --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/tf-train --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/tf-api --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/python-packager --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/tf-train-gpu --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/tf-serve-gpu --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/onnx-serve --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/cluster-autoscaler --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/nvidia --region=$REGISTRY_REGION || true
  aws ecr create-repository --repository-name=cortexlabs/metrics-server --region=$REGISTRY_REGION || true
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

function cleanup() {
  docker container prune -f
  docker image prune -f
}

cmd=${1:-""}
env=${2:-""}

if [ "$cmd" = "create" ]; then
  create_registry

elif [ "$cmd" = "manager" ]; then
  build $ROOT/images/manager manager latest

elif [ "$cmd" = "update" ]; then
  if [ "$env" != "dev" ]; then
    build_and_push $ROOT/images/manager manager latest

    cache_builder $ROOT/images/spark-base spark-base
    build_base $ROOT/images/spark-base spark-base
    build_base $ROOT/images/tf-base tf-base
    build_base $ROOT/images/tf-base-gpu tf-base-gpu

    cache_builder $ROOT/images/operator operator
    build_and_push $ROOT/images/operator operator latest

    cache_builder $ROOT/images/spark-operator spark-operator
    build_and_push $ROOT/images/spark-operator spark-operator latest
    build_and_push $ROOT/images/spark spark latest
    build_and_push $ROOT/images/tf-train tf-train latest
    build_and_push $ROOT/images/tf-train-gpu tf-train-gpu latest
    build_and_push $ROOT/images/nginx-controller nginx-controller latest
    build_and_push $ROOT/images/nginx-backend nginx-backend latest
    build_and_push $ROOT/images/fluentd fluentd latest
    build_and_push $ROOT/images/argo-controller argo-controller latest
    build_and_push $ROOT/images/argo-executor argo-executor latest
    build_and_push $ROOT/images/tf-serve tf-serve latest
    build_and_push $ROOT/images/tf-serve-gpu tf-serve-gpu latest
    build_and_push $ROOT/images/python-packager python-packager latest
    build_and_push $ROOT/images/cluster-autoscaler cluster-autoscaler latest
    build_and_push $ROOT/images/nvidia nvidia latest
    build_and_push $ROOT/images/metrics-server metrics-server latest
  fi

  build_and_push $ROOT/images/tf-api tf-api latest
  build_and_push $ROOT/images/onnx-serve onnx-serve latest

  cleanup
fi
