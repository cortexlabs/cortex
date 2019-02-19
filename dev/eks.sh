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
  eksctl create cluster --version=1.11 --name=$K8S_NAME  --region $K8S_REGION --nodes=$K8S_NODE_COUNT --node-type=$K8S_NODE_INSTANCE_TYPE
  eks_set_cluster

elif [ "$1" = "update" ]; then
  echo "Not implemented"

elif [ "$1" = "stop" ]; then
  eksctl delete cluster --name=$K8S_NAME

elif [ "$1" = "set" ]; then
  eks_set_cluster
fi
