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

eval $(python $ROOT/manager/cluster_config_env.py $ROOT/dev/config/cluster.yaml)

eksctl utils write-kubeconfig --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true

echo "Uninstalling Cortex ..."

kubectl delete --ignore-not-found=true namespace istio-system
kubectl delete --ignore-not-found=true namespace cortex
kubectl delete --ignore-not-found=true -n kube-system deployment cluster-autoscaler
kubectl delete --ignore-not-found=true apiservice v1beta1.metrics.k8s.io
kubectl delete --ignore-not-found=true -n kube-system deployment metrics-server
kubectl delete --ignore-not-found=true -n kube-system service metrics-server
kubectl delete --ignore-not-found=true -n kube-system daemonset istio-cni-node
kubectl delete --ignore-not-found=true -n kube-system daemonset aws-node
kubectl delete --all crds

echo "✓ Uninstalled Cortex"
