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
source $ROOT/dev/config/k8s.sh

function eks_set_cluster() {
  eksctl utils write-kubeconfig --name=$K8S_NAME
  kubectl config set-context $(kubectl config current-context) --namespace="cortex"
}

if [ "$1" = "start" ]; then
  eksctl create cluster --version=1.11 --name=$K8S_NAME  --region $K8S_REGION --nodes-max $K8S_NODES_MAX_COUNT --nodes-min $K8S_NODES_MIN_COUNT --node-type=$K8S_NODE_INSTANCE_TYPE
  if [ $K8S_GPU_NODES_MIN_COUNT -gt 0 ] || [ $K8S_GPU_NODES_MAX_COUNT -gt 0 ]; then
    eksctl create nodegroup --version=1.11 --cluster=$K8S_NAME --nodes-max=$K8S_GPU_NODES_MAX_COUNT --nodes-min=$K8S_GPU_NODES_MIN_COUNT  --node-type=$K8S_GPU_NODE_INSTANCE_TYPE --node-ami=$K8S_GPU_NODE_AMI
    echo "Once the GPU nodegroup joins the cluster, run:"
    echo "kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v1.11/nvidia-device-plugin.yml"
  fi
  eks_set_cluster
  
elif [ "$1" = "update" ]; then
  echo "Not implemented"

elif [ "$1" = "stop" ]; then
  $ROOT/cortex.sh -c=$ROOT/dev/config/cortex.sh uninstall operator 2>/dev/null || true
  eksctl delete cluster --name=$K8S_NAME

elif [ "$1" = "set" ]; then
  eks_set_cluster
fi
