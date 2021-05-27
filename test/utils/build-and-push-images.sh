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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. >/dev/null && pwd)"
source $ROOT/dev/util.sh

# registry address
host=$1

api_images=(
  "async-text-generator-cpu"
  "async-text-generator-gpu"
  "batch-image-classifier-alexnet-cpu"
  "batch-image-classifier-alexnet-gpu"
  "batch-sum-cpu"
  "realtime-image-classifier-resnet50-cpu"
  "realtime-image-classifier-resnet50-gpu"
  "realtime-prime-generator-cpu"
  "realtime-sleep-cpu"
  "realtime-text-generator-cpu"
  "realtime-text-generator-gpu"
  "realtime-hello-world-cpu"
  "task-iris-classifier-trainer-cpu"
)

# build the images
for image in "${api_images[@]}"; do
    kind=$(python -c "first_element='$image'.split('-', 1)[0]; print(first_element)")
    api_name=$(python -c "right_tail='$image'.split('-', 1)[1]; mid_section=right_tail.rsplit('-', 1)[0]; print(mid_section)")
    compute_type=$(python -c "last_element='$image'.rsplit('-', 1)[1]; print(last_element)")
    dir="${ROOT}/test/apis/${kind}/${api_name}"

    blue_echo "Building cortexlabs-test/$image:latest..."
    docker build $dir -f $dir/$api_name-$compute_type.dockerfile -t cortexlabs-test/$image
    green_echo "Built cortexlabs-test/$image:latest\n"
done

# push the images
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
for image in "${api_images[@]}"; do
    blue_echo "Pushing cortexlabs-test/$image:latest..."
    docker push $host/cortexlabs-test/${image}
    green_echo "Pushed cortexlabs-test/$image:latest\n"
done
