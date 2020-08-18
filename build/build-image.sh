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


set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

CORTEX_VERSION=master

slim="false"
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --include-slim)
    slim="true"
    shift
    ;;
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done
set -- "${positional_args[@]}"

dir=$1
image=$2

docker build "$ROOT" \
  -f $dir/Dockerfile \
  -t cortexlabs/${image} \
  -t cortexlabs/${image}:${CORTEX_VERSION}

if [ "$slim" == "true" ]; then
  if [ "${image}" == "python-predictor-gpu" ]; then
    cuda=("10.0" "10.1" "10.2" "11.0")
    cudnn=("7" "7" "7" "8")

    for i in ${!cudnn[@]}; do
      docker build "$ROOT" \
        -f $dir/Dockerfile \
        --build-arg SLIM=true \
        --build-arg CUDA_VERSION=${cuda[$i]} \
        --build-arg CUDNN=${cudnn[$i]} \
        -t cortexlabs/${image}-slim:${CORTEX_VERSION}-cuda${cuda[$i]}
    done
  else
    docker build "$ROOT" \
      -f $dir/Dockerfile \
      --build-arg SLIM=true \
      -t cortexlabs/${image}-slim:${CORTEX_VERSION}
  fi
fi
