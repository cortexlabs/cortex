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

set -eo pipefail

function get_operator_endpoint() {
  kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function get_api_load_balancer_endpoint() {
  kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function get_api_gateway_endpoint() {
  python get_api_gateway_endpoint.py
}

if ! eksctl utils describe-stacks --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION >/dev/null 2>&1; then
  echo "error: there is no cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION; please update your configuration to point to an existing cortex cluster or create a cortex cluster with \`cortex cluster up\`"
  exit 1
fi

eksctl utils write-kubeconfig --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true

operator_endpoint=$(get_operator_endpoint)
api_load_balancer_endpoint=$(get_api_load_balancer_endpoint)
api_gateway_endpoint=$(get_api_gateway_endpoint)

echo -e "\033[1mendpoints:\033[0m"
echo "operator:          $operator_endpoint"  # before modifying this, search for this prefix
echo "api load balancer: $api_load_balancer_endpoint"
echo "api gateway:       $api_gateway_endpoint"
