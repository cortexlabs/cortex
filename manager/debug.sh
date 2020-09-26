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

CORTEX_VERSION_MINOR=master

debug_out_path="$1"
mkdir -p "$(dirname "$debug_out_path")"

if ! eksctl utils describe-stacks --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION >/dev/null 2>&1; then
  echo "error: there is no cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION; please update your configuration to point to an existing cortex cluster or create a cortex cluster with \`cortex cluster up\`"
  exit 1
fi

eksctl utils write-kubeconfig --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true
out=$(kubectl get pods 2>&1 || true); if [[ "$out" == *"must be logged in to the server"* ]]; then echo "error: your aws iam user does not have access to this cluster; to grant access, see https://docs.cortex.dev/v/${CORTEX_VERSION_MINOR}/miscellaneous/security#running-cortex-cluster-commands-from-different-iam-users"; exit 1; fi

echo -n "gathering cluster data"

mkdir -p /cortex-debug/k8s
for resource in pods pods.metrics nodes nodes.metrics daemonsets deployments hpa services virtualservices gateways ingresses configmaps jobs replicasets events; do
  kubectl describe $resource --all-namespaces &>/dev/null > "/cortex-debug/k8s/${resource}"
  kubectl get $resource --all-namespaces &>/dev/null > "/cortex-debug/k8s/${resource}-list"
  echo -n "."
done

mkdir -p /cortex-debug/logs
kubectl get pods --all-namespaces -o json | jq '.items[] | "kubectl logs -n \(.metadata.namespace) \(.metadata.name) --all-containers --timestamps --tail=10000 &>/dev/null > /cortex-debug/logs/\(.metadata.namespace).\(.metadata.name) && echo -n ."' | xargs -n 1 bash -c

kubectl top pods --all-namespaces --containers=true &>/dev/null > "/cortex-debug/k8s/top_pods"
kubectl top nodes &>/dev/null > "/cortex-debug/k8s/top_nodes"

mkdir -p /cortex-debug/aws
aws --region=$CORTEX_REGION autoscaling describe-auto-scaling-groups &>/dev/null > "/cortex-debug/aws/asgs"
echo -n "."
aws --region=$CORTEX_REGION autoscaling describe-scaling-activities &>/dev/null > "/cortex-debug/aws/asg-activities"
echo -n "."

(cd / && tar -czf cortex-debug.tgz cortex-debug)
mv /cortex-debug.tgz $debug_out_path

echo " ✓"
