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

# Note: if namespace is changed, the old namespace will not be deleted

echo
eksctl utils write-kubeconfig --name=$CORTEX_CLUSTER

echo -e "\nUninstalling the Cortex operator ..."

kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true deployment operator >/dev/null 2>&1
kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true daemonset fluentd >/dev/null 2>&1  # Pods in DaemonSets cannot be modified

echo "✓ Uninstalled the Cortex operator"
