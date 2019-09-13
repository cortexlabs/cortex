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

if ! command -v golint >/dev/null 2>&1; then
  echo "golint must be installed"
  exit 1
fi

if ! command -v gofmt >/dev/null 2>&1; then
  echo "gofmt must be installed"
  exit 1
fi

if ! command -v black >/dev/null 2>&1; then
  echo "black must be installed"
  exit 1
fi

go vet "$ROOT/..."

output=$(golint "$ROOT/..." | grep -v "comment" || true)
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

output=$(black --quiet --diff --line-length=100 "$ROOT")
if [[ $output ]]; then
  echo "python files not properly formatted:"
  echo "$output"
  exit 1
fi

# Check for missing license
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "./examples/*" \
! -path "./dev/config/*" \
! -path "./bin/*" \
! -path "./.circleci/*" \
! -path "./.git/*" \
! -name LICENSE \
! -name requirements.txt \
! -name "go.*" \
! -name "*.md" \
! -name ".*" \
! -name "Dockerfile" \
-exec grep -L "Copyright 2019 Cortex Labs, Inc" {} \;)
if [[ $output ]]; then
  echo "File(s) are missing Cortex license:"
  echo "$output"
  exit 1
fi

# Check for trailing whitespace
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -name ".*" \
-exec egrep -l " +$" {} \;)
if [[ $output ]]; then
  echo "File(s) have lines with trailing whitespace:"
  echo "$output"
  exit 1
fi

# Check for missing new line at end of file
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -name ".*" \
-print0 | \
xargs -0 -L1 bash -c 'test "$(tail -c 1 "$0")" && echo "No new line at end of $0"' || true)
if [[ $output ]]; then
  echo "$output"
  exit 1
fi

# Check for multiple new lines at end of file
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -name ".*" \
-print0 | \
xargs -0 -L1 bash -c 'test "$(tail -c 2 "$0")" || echo "Multiple new lines at end of $0"' || true)
if [[ $output ]]; then
  echo "$output"
  exit 1
fi

# Check for new line(s) at beginning of file
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -name ".*" \
-print0 | \
xargs -0 -L1 bash -c 'test "$(head -c 1 "$0")" || echo "New line at beginning of $0"' || true)
if [[ $output ]]; then
  echo "$output"
  exit 1
fi
