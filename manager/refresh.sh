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


if ! eksctl utils describe-stacks --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION >/dev/null 2>&1; then
  # note: if modifying this string, search the codebase for it and change all occurrences
  echo "error: there is no cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION; please update your configuration to point to an existing cortex cluster or create a cortex cluster with \`cortex cluster up\`"
  exit 1
fi

eksctl utils write-kubeconfig --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true

kubectl get -n=cortex configmap cluster-config -o yaml >> cluster_configmap.yaml
python refresh_cluster_config.py cluster_configmap.yaml /.cortex/cluster.yaml

# TODO use different name for each cluster, <cluster_name>_<region>.yaml

kubectl -n=cortex create configmap 'cluster-config' \
    --from-file='cluster.yaml'='/.cortex/cluster.yaml' \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
