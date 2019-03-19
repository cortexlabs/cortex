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

# replace examples/iris/app.yaml -> examples/*/app.yaml
for EXAMPLE in $ROOT/examples/*/app.yaml; do
  TIMER=1200
  EXAMPLE_ROOT=$(dirname "${EXAMPLE}")
  RETRY="false"

  cd $EXAMPLE_ROOT
  echo "Deploying $EXAMPLE_ROOT"
  $CORTEX delete >/dev/null
  $CORTEX deploy

API_NAMES="$($CORTEX get api | sed '1,2d' | sed '/^$/d' | tr -s ' ' | cut -f 1 -d " ")"
  echo $API_NAMES
  SAMPLE="$(find . -name "*.json")"

  while true
  do
    $CORTEX status
    CURRENT_STATUS="$($CORTEX status)"

    ERROR_COUNT="$(echo $CURRENT_STATUS | grep "error" | wc -l)"
    if [ $ERROR_COUNT -gt "0" ] && [[ ! $CURRENT_STATUS =~ ^error\:.* ]]; then
      exit 1
    fi

    READY_COUNT="$($CORTEX get api | sed '1,2d' | sed '/^$/d' | grep "ready" | wc -l)"
    TOTAL_COUNT="$($CORTEX get api | sed '1,2d' | sed '/^$/d' | wc -l)"

    sleep 15 # account for api startup delay

    if [ "$READY_COUNT" == "$TOTAL_COUNT" ] && [ $TOTAL_COUNT -ne "0" ]; then
      for API_NAME in $API_NAMES; do
        echo "Running cx predict $API_NAME $SAMPLE"
        $CORTEX predict $API_NAME $SAMPLE
        RESULT="$($CORTEX predict $API_NAME $SAMPLE)"
        if [ $? -ne 0 ]; then
          if [[ $RESULT =~ ^error\:\ failed\ to\ connect\ to\ the\ operator.* ]] || [[ $RESULT =~ ^error\:\ api.*is\ updating$ ]]; then
              echo "retrying prediction..."
              $RETRY="true"
              break
          else
              echo "prediction failed"
              exit 1
          fi
        fi
      done

      if [ "$RETRY" == "false" ]; then
        RETRY="false"
        break
      else
        RETRY="false"
      fi
    fi

    TIMER=$((TIMER-15))
    echo "Running $EXAMPLE_ROOT. $TIMER seconds left before timing out."
    if [ $TIMER -lt "0" ]; then
        echo "timed out!"
        exit 1
    fi
  done

  $CORTEX delete
done
