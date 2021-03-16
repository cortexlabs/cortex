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

if ! eksctl utils describe-stacks --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION >/dev/null 2>&1; then
  echo "error: there is no cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION; please update your configuration to point to an existing cortex cluster or create a cortex cluster with \`cortex cluster up\`"
  exit 1
fi

eksctl utils write-kubeconfig --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION | (grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true)
out=$(kubectl get pods 2>&1 || true); if [[ "$out" == *"must be logged in to the server"* ]]; then echo "error: your aws iam user does not have access to this cluster; to grant access, see https://docs.cortex.dev/v/${CORTEX_VERSION_MINOR}/"; exit 1; fi

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

mkdir -p /cortex-debug/aws/amis

aws autoscaling describe-auto-scaling-groups --region=$CORTEX_REGION --output json > "/cortex-debug/aws/asgs" 2>&1
echo -n "."
aws autoscaling describe-scaling-activities --max-items 1000 --region=$CORTEX_REGION --output json > "/cortex-debug/aws/asg-activities" 2>&1
echo -n "."

aws ec2 describe-instances --filters Name=tag:cortex.dev/cluster-name,Values=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION --output json > "/cortex-debug/aws/instances" 2>&1
echo -n "."
aws ec2 describe-instance-status --include-all-instances --region=$CORTEX_REGION --output json > "/cortex-debug/aws/instance-statuses" 2>&1
echo -n "."
aws ec2 describe-instances --filters Name=tag:cortex.dev/cluster-name,Values=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION --output json | jq "[.Reservations[].Instances[].ImageId] | unique | .[] | \"aws ec2 describe-images --image-ids \(.) --region=$CORTEX_REGION --output json > /cortex-debug/aws/amis/\(.) 2>&1\"" | xargs -n 1 bash -c
echo -n "."
python get_operator_load_balancer_state.py > "/cortex-debug/aws/operator_load_balancer_state" 2>&1
echo -n "."
python get_api_load_balancer_state.py > "/cortex-debug/aws/api_load_balancer_state" 2>&1
echo -n "."
python get_operator_target_group_status.py > "/cortex-debug/aws/operator_load_balancer_target_group_status" 2>&1
echo -n "."

mkdir -p /cortex-debug/misc
operator_endpoint=$(kubectl -n=istio-system get service ingressgateway-operator -o json 2>/dev/null | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
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
