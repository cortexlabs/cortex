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

# if parallel utility is installed, the docker push commands will be parallelized
if command -v parallel &> /dev/null && [ -n "${NUM_BUILD_PROCS+set}" ] && [ "$NUM_BUILD_PROCS" != "1" ]; then
  ROOT=$ROOT DOCKER_USERNAME=$DOCKER_USERNAME DOCKER_PASSWORD=$DOCKER_PASSWORD SHELL=$(type -p /bin/bash) parallel --will-cite --halt now,fail=1 --eta --jobs $NUM_BUILD_PROCS $ROOT/build/push-image.sh {} ::: "${all_images[@]}"
else
  for image in "${all_images[@]}"; do
    $ROOT/build/push-image.sh $image
  done
fi
