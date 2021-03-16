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

set +e

CORTEX_VERSION_MINOR=0.31

debug_out_path="$1"
mkdir -p "$(dirname "$debug_out_path")"

gcloud auth activate-service-account --key-file $GOOGLE_APPLICATION_CREDENTIALS 2> /dev/stdout 1> /dev/null | (grep -v "Activated service account credentials" || true)
gcloud container clusters get-credentials $CORTEX_CLUSTER_NAME --project $CORTEX_GCP_PROJECT --region $CORTEX_GCP_ZONE 2> /dev/stdout 1> /dev/null | (grep -v "Fetching cluster" | grep -v "kubeconfig entry generated" || true)
out=$(kubectl get pods 2>&1 || true); if [[ "$out" == *"must be logged in to the server"* ]]; then echo "error: your iam user does not have access to this cluster"; exit 1; fi

echo -n "gathering cluster data"

mkdir -p /cortex-debug/k8s
for resource in pods pods.metrics nodes nodes.metrics daemonsets deployments hpa services virtualservices gateways ingresses configmaps jobs replicasets events; do
  kubectl describe $resource --all-namespaces > "/cortex-debug/k8s/${resource}" 2>&1
  kubectl get $resource --all-namespaces > "/cortex-debug/k8s/${resource}-list" 2>&1
  echo -n "."
done

mkdir -p /cortex-debug/logs
kubectl get pods --all-namespaces -o json | jq '.items[] | . as $parent | $parent.spec.containers[]? | "kubectl logs -n \($parent.metadata.namespace) \($parent.metadata.name) \(.name) --timestamps --tail=10000 > /cortex-debug/logs/\($parent.metadata.namespace).\($parent.metadata.name).\(.name) 2>&1 && echo -n ."' | xargs -n 1 bash -c
echo -n "."
kubectl get pods --all-namespaces -o json | jq '.items[] | . as $parent | $parent.spec.initContainers[]? | "kubectl logs -n \($parent.metadata.namespace) \($parent.metadata.name) \(.name) --timestamps --tail=10000 > /cortex-debug/logs/\($parent.metadata.namespace).\($parent.metadata.name).init.\(.name) 2>&1 && echo -n ."' | xargs -n 1 bash -c
echo -n "."

kubectl top pods --all-namespaces --containers=true > "/cortex-debug/k8s/top_pods" 2>&1
echo -n "."
kubectl top nodes > "/cortex-debug/k8s/top_nodes" 2>&1
echo -n "."

mkdir -p /cortex-debug/misc
operator_endpoint=$(kubectl -n=istio-system get service ingressgateway-operator -o json 2>/dev/null | tr -d '[:space:]' | sed 's/.*{\"ip\":\"\(.*\)\".*/\1/')
echo "$operator_endpoint" > /cortex-debug/misc/operator_endpoint
if [ "$operator_endpoint" == "" ]; then
  echo "unable to get operator endpoint" > /cortex-debug/misc/operator_curl
else
  curl -sv --max-time 5 "${operator_endpoint}/verifycortex" > /cortex-debug/misc/operator_curl 2>&1
fi
echo -n "."

(cd / && tar -czf cortex-debug.tgz cortex-debug)
mv /cortex-debug.tgz $debug_out_path

echo " ✓"
