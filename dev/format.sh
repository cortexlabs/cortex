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

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

function run_go_format() {
  if ! command -v gofmt >/dev/null 2>&1; then
    echo "gofmt must be installed"
    exit 1
  fi

  gofmt -s -w "$ROOT"
}

function run_python_format() {
  if ! command -v black >/dev/null 2>&1; then
    echo "black must be installed"
    exit 1
  fi

  black --quiet --line-length=100 "$ROOT"
}

cmd=${1:-""}

if [ "$cmd" = "go" ]; then
  run_go_format
elif [ "$cmd" = "python" ]; then
  run_python_format
else
  run_go_format
  run_python_format
fi
