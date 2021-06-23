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

source $ROOT/build/images.sh
source $ROOT/dev/util.sh

arch=$1
host_primary=$2
host_backup=$3

for image in "${all_images[@]}"; do
  is_multi_arch="false"
  if in_array $image "multi_arch_images"; then
    is_multi_arch="true"
    $ROOT/build/push-image.sh $host_primary $host_backup $image $is_multi_arch $arch
  elif [ "$arch" = "amd64" ]; then
    $ROOT/build/push-image.sh $host_primary $host_backup $image $is_multi_arch $arch
  fi
done
