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

image=$1
image_type=$2

if [ "$image" == "inferentia" ]; then
  aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin 790709498068.dkr.ecr.us-west-2.amazonaws.com
fi

if [ "$image_type" = "non-api" ]; then
  docker build "$ROOT" -f $ROOT/images/$image/Dockerfile $build_args -t quay.io/cortexlabs/${image}:${CORTEX_VERSION} -t cortexlabs/${image}:${CORTEX_VERSION}
else
  num_words="$(awk -F'-' '{ for(i=1;i<=NF;i++) print $i }' <<< ${image} | wc -l | sed -e 's/^[[:space:]]*//')"
  kind_dir="$( cut -d '-' -f 1 <<< "${image}" )"
  api_name="$( cut -d '-' -f 2-$((num_words-1)) <<< "${image}" )"
  compute_type="$( cut -d '-' -f ${num_words} <<< "${image}" )"

  dir="${ROOT}/test/apis/${kind_dir}/${api_name}"
  docker build $dir -f $dir/$api_name-$compute_type.dockerfile -t cortexlabs/$image:${CORTEX_VERSION}
fi
