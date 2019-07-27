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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"
PWD=$(pwd)
source $ROOT/dev/config/cortex.sh


function uninstall_cortex() {
  $ROOT/dev/uninstall_cortex.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER=$CORTEX_CLUSTER \
    -e CORTEX_REGION=$CORTEX_REGION \
    -e CORTEX_NAMESPACE=$CORTEX_NAMESPACE \
    $CORTEX_IMAGE_MANAGER
}

function info() {
   docker run -it --entrypoint ./info.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER=$CORTEX_CLUSTER \
    -e CORTEX_REGION=$CORTEX_REGION \
    -e CORTEX_NAMESPACE=$CORTEX_NAMESPACE \
    $CORTEX_IMAGE_MANAGER
}


function install_cortex() {
   docker run -it --entrypoint ./install_cortex.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER=$CORTEX_CLUSTER \
    -e CORTEX_REGION=$CORTEX_REGION \
    -e CORTEX_NAMESPACE=$CORTEX_NAMESPACE \
    -e CORTEX_NODE_TYPE=$CORTEX_NODE_TYPE \
    -e CORTEX_LOG_GROUP=$CORTEX_LOG_GROUP \
    -e CORTEX_BUCKET=$CORTEX_BUCKET \
    -e CORTEX_IMAGE_FLUENTD=$CORTEX_IMAGE_FLUENTD \
    -e CORTEX_IMAGE_OPERATOR=$CORTEX_IMAGE_OPERATOR \
    -e CORTEX_IMAGE_SPARK=$CORTEX_IMAGE_SPARK \
    -e CORTEX_IMAGE_SPARK_OPERATOR=$CORTEX_IMAGE_SPARK_OPERATOR \
    -e CORTEX_IMAGE_TF_SERVE=$CORTEX_IMAGE_TF_SERVE \
    -e CORTEX_IMAGE_TF_TRAIN=$CORTEX_IMAGE_TF_TRAIN \
    -e CORTEX_IMAGE_TF_API=$CORTEX_IMAGE_TF_API \
    -e CORTEX_IMAGE_PYTHON_PACKAGER=$CORTEX_IMAGE_PYTHON_PACKAGER \
    -e CORTEX_IMAGE_TF_SERVE_GPU=$CORTEX_IMAGE_TF_SERVE_GPU \
    -e CORTEX_IMAGE_TF_TRAIN_GPU=$CORTEX_IMAGE_TF_TRAIN_GPU \
    -e CORTEX_IMAGE_ONNX_SERVE=$CORTEX_IMAGE_ONNX_SERVE \
    -e CORTEX_IMAGE_ONNX_SERVE_GPU=$CORTEX_IMAGE_ONNX_SERVE_GPU \
    -e CORTEX_IMAGE_CLUSTER_AUTOSCALER=$CORTEX_IMAGE_CLUSTER_AUTOSCALER \
    -e CORTEX_IMAGE_NVIDIA=$CORTEX_IMAGE_NVIDIA \
    -e CORTEX_IMAGE_METRICS_SERVER=$CORTEX_IMAGE_METRICS_SERVER \
    -e CORTEX_IMAGE_ISTIO_CITADEL=$CORTEX_IMAGE_ISTIO_CITADEL \
    -e CORTEX_IMAGE_ISTIO_GALLEY=$CORTEX_IMAGE_ISTIO_GALLEY \
    -e CORTEX_IMAGE_ISTIO_SIDECAR=$CORTEX_IMAGE_ISTIO_SIDECAR \
    -e CORTEX_IMAGE_ISTIO_PILOT=$CORTEX_IMAGE_ISTIO_PILOT \
    -e CORTEX_ENABLE_TELEMETRY=$CORTEX_ENABLE_TELEMETRY \
    $CORTEX_IMAGE_MANAGER
}

cd $ROOT/manager
arg1=${1:-""}
if [ "$arg1" = "install" ]; then
  install_cortex && info
elif [ "$arg1" = "uninstall" ]; then
  uninstall_cortex
fi
cd $PWD
