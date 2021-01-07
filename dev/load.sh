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


SLEEP="0"  # seconds

# Iris
URL="http://a4ee971c803e211ea9dd806fb935f8b0-108090235.us-west-2.elb.amazonaws.com/iris/classifier"
DATA='{ "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 }'

trap ctrl_c INT
function ctrl_c() {
  echo ""
  exit 0
}

function make_request() {
  curl --silent --show-error -k -X POST -H "Content-Type: application/json" -d "${DATA}" "${URL}"
}

resp=$(make_request)
echo -n "."

while eval "sleep ${SLEEP}"; do
  resp=$(make_request)
  echo -n "."
done
