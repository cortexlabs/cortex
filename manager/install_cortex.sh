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

function validate_cortex() {
  set +e

  echo -en "\nWaiting for Cortex to be ready "

  operator_load_balancer="waiting"
  api_load_balancer="waiting"
  operator_endpoint_reachable="waiting"
  operator_pod_ready_cycles=0
  operator_endpoint=""

  while true; do
    echo -n "."
    sleep 5

    operator_pod_name=$(kubectl -n=$CORTEX_NAMESPACE get pods -o=name --sort-by=.metadata.creationTimestamp | grep "^pod/operator-" | tail -1)
    if [ "$operator_pod_name" == "" ]; then
      operator_pod_ready_cycles=0
    else
      is_ready=$(kubectl -n=$CORTEX_NAMESPACE get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].ready}')
      if [ "$is_ready" == "true" ]; then
        ((operator_pod_ready_cycles++))
      else
        operator_pod_ready_cycles=0
      fi
    fi

    if [ "$operator_load_balancer" != "ready" ]; then
      out=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-operator -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        continue
      fi
      operator_load_balancer="ready"
    fi

    if [ "$api_load_balancer" != "ready" ]; then
      out=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-apis -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        continue
      fi
      api_load_balancer="ready"
    fi

    if [ "$operator_endpoint" = "" ]; then
      operator_endpoint=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
    fi

    if [ "$operator_endpoint_reachable" != "ready" ]; then
      if ! curl $operator_endpoint >/dev/null 2>&1; then
        continue
      fi
      operator_endpoint_reachable="ready"
    fi

    if [ "$operator_pod_ready_cycles" == "0" ] && [ "$operator_pod_name" != "" ]; then
      num_restart=$(kubectl -n=$CORTEX_NAMESPACE get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].restartCount}')
      if [[ $num_restart -ge 2 ]]; then
        echo -e "\n\nAn error occurred when starting the Cortex operator. View the logs with:"
        echo "  kubectl logs $operator_pod_name --namespace=$CORTEX_NAMESPACE"
        exit 1
      fi
      continue
    fi

    if [[ $operator_pod_ready_cycles -lt 3 ]]; then
      continue
    fi

    break
  done

  echo -e "\n\n✓ Cortex is ready!"

  if command -v cortex >/dev/null; then
    echo -e "\nPlease run \`cortex configure\` to make sure your CLI is configured correctly"
  fi
}

echo
eksctl utils write-kubeconfig --name=$CORTEX_CLUSTER

echo -e "\nInstalling Cortex ..."

envsubst < manifests/namespace.yaml | kubectl apply -f - >/dev/null
envsubst < manifests/spark.yaml | kubectl apply -f - >/dev/null
envsubst < manifests/argo.yaml | kubectl apply -f - >/dev/null
envsubst < manifests/nginx.yaml | kubectl apply -f - >/dev/null
envsubst < manifests/fluentd.yaml | kubectl apply -f - >/dev/null
envsubst < manifests/operator.yaml | kubectl apply -f - >/dev/null

validate_cortex
