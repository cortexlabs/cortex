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

echo
eksctl utils write-kubeconfig --name=$CORTEX_CLUSTER

echo -e "\nUninstalling Cortex ..."

# Remove finalizers on sparkapplications (they sometimes create deadlocks)
if kubectl get namespace $CORTEX_NAMESPACE >/dev/null 2>&1 && kubectl get customresourcedefinition sparkapplications.sparkoperator.k8s.io >/dev/null 2>&1; then
  set +e
  kubectl -n=$CORTEX_NAMESPACE get sparkapplications.sparkoperator.k8s.io -o name | xargs -L1 \
    kubectl -n=$CORTEX_NAMESPACE patch -p '{"metadata":{"finalizers": []}}' --type=merge >/dev/null 2>&1
  set -e
fi

kubectl delete --ignore-not-found=true customresourcedefinition scheduledsparkapplications.sparkoperator.k8s.io >/dev/null 2>&1
kubectl delete --ignore-not-found=true customresourcedefinition sparkapplications.sparkoperator.k8s.io >/dev/null 2>&1
kubectl delete --ignore-not-found=true customresourcedefinition workflows.argoproj.io >/dev/null 2>&1
kubectl delete --ignore-not-found=true namespace $CORTEX_NAMESPACE >/dev/null 2>&1

echo "✓ Uninstalled Cortex"
