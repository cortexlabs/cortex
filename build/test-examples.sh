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

set -eou pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"
CORTEX="$ROOT/bin/cortex"

# replace examples/iris/app.yaml -> examples/*/app.yaml
for example in $ROOT/examples/*/app.yaml; do
  timer=1200
  example_base_dir=$(dirname "${example}")
  retry="false"

  cd $example_base_dir
  echo "Deploying $example_base_dir"
  $CORTEX delete || test $? = 1
  $CORTEX deploy

  api_names="$($CORTEX get api | sed '1,2d' | sed '/^$/d' | tr -s ' ' | cut -f 1 -d " ")"
  sample="$(find . -name "*.json")"

  while true
  do
    # $CORTEX status
    current_status="$($CORTEX status)"
    echo "$current_status"

    error_count="$(echo $current_status | { grep "error" || test $? = 1; } | wc -l)"
    if [ $error_count -gt "0" ] && [[ ! $current_status =~ ^error\:\ failed\ to\ connect\ to\ the\ operator.* ]]; then
      exit 1
    fi

    ready_count="$($CORTEX get api | sed '1,2d' | sed '/^$/d' | { grep "ready" || test $? = 1; } | wc -l)"
    total_count="$($CORTEX get api | sed '1,2d' | sed '/^$/d' | wc -l)"

    sleep 15 # account for api startup delay

    if [ "$ready_count" == "$total_count" ] && [ $total_count -ne "0" ]; then
      for api_name in $api_names; do
        echo "Running cx predict $api_name $sample"
        result="$($CORTEX predict $api_name $sample)"
        echo "$result"
        if [ $? -ne 0 ]; then
          if [[ $result =~ ^error\:\ failed\ to\ connect\ to\ the\ operator.* ]] || [[ $result =~ ^error\:\ api.*is\ updating$ ]]; then
              echo "retrying prediction..."
              $retry="true"
              break
          else
              echo "prediction failed"
              exit 1
          fi
        fi
      done

      if [ "$retry" == "false" ]; then
        break
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
