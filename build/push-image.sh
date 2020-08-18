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

image=$1

echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

docker push cortexlabs/${image}:${CORTEX_VERSION}

if [ "$slim" == "true" ]; then
  if [ "$image" == "python-predictor-gpu" ]; then
    for cuda in 10.0 10.1 10.2 11.0; do
      docker push cortexlabs/${image}-slim:${CORTEX_VERSION}-cuda${cuda}
    done
  else
    docker push cortexlabs/${image}-slim:${CORTEX_VERSION}
  fi
fi
