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


set -euo pipefail

CORTEX_VERSION=master

host_primary=$1
host_backup=$2
image=$3
has_arm64_build=$4

hosts=(
    "$host_primary"
    "$host_backup"
)
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

for host in "${hosts[@]}"; do
    if [ "$has_arm64_build" = "true" ]; then
        docker manifest create $host/cortexlabs/${image}:${CORTEX_VERSION} \
            -a $host/cortexlabs/${image}:manifest-${CORTEX_VERSION}-amd64 \
            -a $host/cortexlabs/${image}:manifest-${CORTEX_VERSION}-arm64
    else
        docker manifest create $host/cortexlabs/${image}:${CORTEX_VERSION} \
            -a $host/cortexlabs/${image}:manifest-${CORTEX_VERSION}-amd64
    fi
    docker manifest push $host/cortexlabs/${image}:${CORTEX_VERSION}
done
