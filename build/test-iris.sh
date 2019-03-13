#!/bin/bash

# Copyright 2019 Cortex Labs, Inc.
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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"
CORTEX="$ROOT/bin/cortex"

for EXAMPLES in $ROOT/examples/iris/app.yaml; do
  timer=1200
  EXAMPLE_ROOT=$(dirname "${EXAMPLES}")


  cd $EXAMPLE_ROOT
  echo "Deploying $EXAMPLE_ROOT"
  $CORTEX delete >/dev/null
  $CORTEX deploy
  
  API_NAME="$($CORTEX get api | awk '/NAME/{getline; print}' | cut -f 1 -d " ")"
  SAMPLE="$(find . -name "*.json")"

  while true 
  do
    $CORTEX status

    # check for errors
    ERROR_COUNT="$($CORTEX status | grep "error" | wc -l)"
    if [ $ERROR_COUNT -gt "0" ]; then
      exit 1
    fi

    # if api is ready request for predictions
    API_READY="$($CORTEX get api | grep "$API_NAME" | grep "ready" | wc -l)"
    sleep 5 # account for api startup delay
    if [ $API_READY -gt "0" ]; then
      $CORTEX predict $API_NAME $SAMPLE
      if [ $? -eq 0 ]; then
        break
      else
        echo "prediction failed"
        exit 1
      fi
    fi

    sleep 10
    timer=$((timer-15))
    echo "Running $EXAMPLE_ROOT. $timer seconds left before timing out."
    if [ $timer -lt "0" ]; then
        echo "timed out!"
        exit 1
    fi
  done

  $CORTEX delete
done
