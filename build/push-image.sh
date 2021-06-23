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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

CORTEX_VERSION=master

host_primary=$1
host_backup=$2
image=$3
platforms=$4

echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
docker buildx build $ROOT --cache-to=type=local,src=$ROOT/.cache --progress plain -f $ROOT/images/$image/Dockerfile -t $host_primary/cortexlabs/${image}:${CORTEX_VERSION} -t $host_backup/cortexlabs/${image}:${CORTEX_VERSION} --platform $platforms --push
