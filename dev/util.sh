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


function blue_echo() {
  echo -e "\033[1;34m$1\033[0m"
}

function green_echo() {
  echo -e "\033[1;32m$1\033[0m"
}

function error_echo() {
  echo -e "\033[1;31mERROR: \033[0m$1"
}

function join_by() {
  # Argument #1 is the separator. It can be multi-character.
  # Argument #2, 3, and so on, are the elements to be joined.
  # Usage: join_by ", " "${array[@]}"
  local SEPARATOR="$1"
  shift

  local F=0
  for x in "$@"; do
    if [[ F -eq 1 ]]; then
      echo -n "$SEPARATOR"
    else
      F=1
    fi
    echo -n "$x"
  done
  echo
}
