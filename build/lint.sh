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

function run_go_lint() {
  if ! command -v golint >/dev/null 2>&1; then
    echo "golint must be installed"
    exit 1
  fi

  if ! command -v gofmt >/dev/null 2>&1; then
    echo "gofmt must be installed"
    exit 1
  fi

  output=$(go vet "$ROOT/..." 2>&1)
  if [[ $output ]]; then
    echo "$output"
    exit 1
  fi

  output=$(golint "$ROOT/..." | grep -v "comment")
  if [[ $output ]]; then
    echo "$output"
    exit 1
  fi

  output=$(gofmt -s -l "$ROOT")
  if [[ $output ]]; then
    echo "go files not properly formatted:"
    echo "$output"
    exit 1
  fi
}

function run_python_lint() {
  if ! command -v black >/dev/null 2>&1; then
    echo "black must be installed"
    exit 1
  fi

  output=$(black --quiet --diff --line-length=100 "$ROOT")
  if [[ $output ]]; then
    echo "python files not properly formatted:"
    echo "$output"
    exit 1
  fi
}

cmd=${1:-""}

if [ "$cmd" = "go" ]; then
  run_go_lint
elif [ "$cmd" = "python" ]; then
  run_python_lint
else
  run_go_lint
  run_python_lint
fi
