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

echo -e "spinning down the cluster ...\n"

eksctl delete cluster --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION

echo -e "\n✓ spun down the cluster"
echo "✓ eks cloudformation stack (eksctl-${CORTEX_CLUSTER_NAME}-cluster) deletion in progress: https://${CORTEX_REGION}.console.aws.amazon.com/cloudformation"
