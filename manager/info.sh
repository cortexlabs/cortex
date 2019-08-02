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

set -e

function get_operator_endpoint() {
  set -eo pipefail
  kubectl -n=istio-system get service operator-ingressgateway -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function get_apis_endpoint() {
  set -eo pipefail
  kubectl -n=istio-system get service apis-ingressgateway -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

eksctl utils write-kubeconfig --name=$CORTEX_CLUSTER --region=$CORTEX_REGION | grep -v "saved kubeconfig as" || true

operator_endpoint=$(get_operator_endpoint)
apis_endpoint=$(get_apis_endpoint)

echo "Operator endpoint:  $operator_endpoint"
echo "APIs endpoint:      $apis_endpoint"

echo -e "\nRun \`cortex configure\` to configure your CLI"
