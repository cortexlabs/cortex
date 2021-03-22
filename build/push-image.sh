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

CORTEX_VERSION=0.31.1

image=$1

echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

if [ "$image" == "python-predictor-gpu" ]; then
  cuda=("10.0" "10.1" "10.1" "10.2" "10.2" "11.0" "11.1")
  cudnn=("7" "7" "8" "7" "8" "8" "8")
  for i in ${!cudnn[@]}; do
    docker push quay.io/cortexlabs/${image}:${CORTEX_VERSION}-cuda${cuda[$i]}-cudnn${cudnn[$i]}
  done
else
  docker push quay.io/cortexlabs/${image}:${CORTEX_VERSION}
fi
