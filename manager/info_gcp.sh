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

CORTEX_VERSION_MINOR=0.31

function get_operator_endpoint() {
  kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]' | sed 's/.*{\"ip\":\"\(.*\)\".*/\1/'
}

function get_api_load_balancer_endpoint() {
  kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]' | sed 's/.*{\"ip\":\"\(.*\)\".*/\1/'
}

gcloud auth activate-service-account --key-file $GOOGLE_APPLICATION_CREDENTIALS 2> /dev/stdout 1> /dev/null | (grep -v "Activated service account credentials" || true)
gcloud container clusters get-credentials $CORTEX_CLUSTER_NAME --project $CORTEX_GCP_PROJECT --region $CORTEX_GCP_ZONE 2> /dev/stdout 1> /dev/null | (grep -v "Fetching cluster" | grep -v "kubeconfig entry generated" || true)
out=$(kubectl get pods 2>&1 || true); if [[ "$out" == *"must be logged in to the server"* ]]; then echo "error: your iam user does not have access to this cluster"; exit 1; fi

operator_endpoint=$(get_operator_endpoint)
api_load_balancer_endpoint=$(get_api_load_balancer_endpoint)

echo -e "\033[1mendpoints:\033[0m"
echo "operator:          $operator_endpoint"  # before modifying this, search for this prefix
echo "api load balancer: $api_load_balancer_endpoint"
