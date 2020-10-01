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

source $ROOT/build/images.sh
source $ROOT/dev/util.sh

if command -v parallel &> /dev/null ; then
  images=$(join_by "," "${ci_images[@]}")
  ROOT=$ROOT SHELL=$(type -p /bin/bash) parallel --eta -k -d"," --colsep=" " $ROOT/build/build-image.sh images/{1} {1} {2} ::: "${images}"
else
  MAX=$(echo $images | wc -w)
  for i in `seq 1 ${MAX}`; do
    image=$(echo $images | cut -d " " -f $i)
    option=$(echo $options | cut -d " " -f $i)
    $ROOT/build/build-image.sh images/$image $image $option
  done
fi
