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

ECR_LOGGED_IN=false

function ecr_login() {
  if [ "$ECR_LOGGED_IN" = false ]; then
    blue_echo "Logging in to ECR..."
    ecr_login_command=$(aws ecr get-login --no-include-email --region $REGISTRY_REGION)
    eval $ecr_login_command
    ECR_LOGGED_IN=true
    green_echo "Success\n"
  fi
}

function create_registry() {
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
}

### HELPERS ###

function build() {
  DIR=$1
  IMAGE=$2
  TAG=$3

  blue_echo "Building $IMAGE:$TAG..."
  docker build $ROOT -f $DIR/Dockerfile -t cortexlabs/$IMAGE:$TAG -t $REGISTRY_URL/cortexlabs/$IMAGE:$TAG
  green_echo "Built $IMAGE:$TAG\n"
}

function build_base() {
  DIR=$1
  IMAGE=$2

  blue_echo "Building $IMAGE..."
  docker build $ROOT -f $DIR/Dockerfile -t cortexlabs/$IMAGE:latest
  green_echo "Built $IMAGE\n"
}

function cache_builder() {
  DIR=$1
  IMAGE=$2

  blue_echo "Building $IMAGE-builder..."
  docker build $ROOT -f $DIR/Dockerfile -t cortexlabs/$IMAGE-builder:latest --target builder
  green_echo "Built $IMAGE-builder\n"
}

function push() {
  ecr_login

  IMAGE=$1
  TAG=$2

  blue_echo "Pushing $IMAGE:$TAG..."
  docker push $REGISTRY_URL/cortexlabs/$IMAGE:$TAG
  green_echo "Pushed $IMAGE:$TAG\n"
}

function build_and_push() {
  DIR=$1
  IMAGE=$2
  TAG=$3

  build $DIR $IMAGE $TAG
  push $IMAGE $TAG
}

function cleanup() {
  docker container prune -f
  docker image prune -f
}

CMD=${1:-""}
ENV=${2:-""}

if [ "$CMD" = "create" ]; then
  create_registry

elif [ "$CMD" = "update" ]; then
  if [ "$ENV" != "dev" ]; then
    cache_builder $ROOT/images/spark-base spark-base
    build_base $ROOT/images/spark-base spark-base
    build_base $ROOT/images/tf-base tf-base

    cache_builder $ROOT/images/operator operator
    build_and_push $ROOT/images/operator operator latest

    cache_builder $ROOT/images/spark-operator spark-operator
    build_and_push $ROOT/images/spark-operator spark-operator latest

    build_and_push $ROOT/images/nginx-controller nginx-controller latest
    build_and_push $ROOT/images/nginx-backend nginx-backend latest
    build_and_push $ROOT/images/fluentd fluentd latest
    build_and_push $ROOT/images/argo-controller argo-controller latest
    build_and_push $ROOT/images/argo-executor argo-executor latest
    build_and_push $ROOT/images/tf-serve tf-serve latest
    build_and_push $ROOT/images/python-packager python-packager latest
  fi

  build_and_push $ROOT/images/spark spark latest
  build_and_push $ROOT/images/tf-train tf-train latest
  build_and_push $ROOT/images/tf-api tf-api latest

  cleanup
fi
