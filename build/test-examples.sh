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

set -eou pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"
CORTEX="$ROOT/bin/cortex"

for example in $ROOT/test/*/cortex.yaml; do
  timer=1200
  example_base_dir=$(dirname "${example}")
  retry="false"

  cd $example_base_dir
  echo "Deploying $example_base_dir"
  $CORTEX deploy

  api_names="$($CORTEX get | sed '1,2d' | sed '/^$/d' | tr -s ' ' | cut -f 1 -d " ")"
  payload="$(find . -name "*.json")"

  while true; do
    current_status="$($CORTEX status)"
    echo "$current_status"

    error_count="$(echo $current_status | { grep "error" || test $? = 1; } | wc -l)"
    # accommodate transient error `error: failed to connect to operator...`
    if [ $error_count -gt "0" ] && [[ ! $current_status =~ ^error\:\ failed\ to\ connect\ to\ the\ operator.* ]]; then
      exit 1
    fi

    ready_count="$($CORTEX get | sed '1,2d' | sed '/^$/d' | { grep "ready" || test $? = 1; } | wc -l)"
    total_count="$($CORTEX get | sed '1,2d' | sed '/^$/d' | wc -l)"

    sleep 15 # account for API startup delay

    if [ "$ready_count" == "$total_count" ] && [ $total_count -ne "0" ]; then
      for api_name in $api_names; do
        echo "Running cx predict $api_name $payload"
        result="$($CORTEX predict $api_name $payload)"
        prediction_exit_code=$?
        echo "$result"
        if [ $prediction_exit_code -ne 0 ]; then
          # accommodate transient error `error: failed to connect to operator...`
          # handle `error: api is updating` error caused when the API status is set to `ready` but it actually isn't
          if [[ $result =~ ^error\:\ failed\ to\ connect\ to\ the\ operator.* ]] || [[ $result =~ ^error\:\ .*is\ updating$ ]]; then
              echo "retrying prediction..."
              $retry="true"
              break # skip request predictions from the remaining APIs and try again
          else
              echo "prediction failed"
              exit 1
          fi
        fi
      done

      if [ "$retry" == "false" ]; then
        break # successfully got predictions from all APIs for this example, move into the next
      else
        retry="false"
      fi
    fi

    timer=$((timer-15))
    echo "Running $example_base_dir. $timer seconds left before timing out."
    if [ $timer -lt "0" ]; then
        echo "timed out!"
        exit 1
    fi
  done

  $CORTEX delete
done

echo "Ran all examples successfully."
