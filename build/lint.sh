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


set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

git_branch="${CIRCLE_BRANCH:-""}"
if [ "$git_branch" = "" ]; then
  git_branch=$(cd "$ROOT" && git rev-parse --abbrev-ref HEAD)
fi

is_release_branch="false"
if echo "$git_branch" | grep -Eq ^[0-9]+.[0-9]+$; then
  is_release_branch="true"
fi

if ! command -v golint >/dev/null 2>&1; then
  echo "golint must be installed"
  exit 1
fi

if ! command -v looppointer >/dev/null 2>&1; then
  echo "looppointer must be installed"
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

output=$(looppointer "$ROOT/...")
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
  black --version
  exit 1
fi

# Check for missing license
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "**/.vscode/*" \
! -path "**/.idea/*" \
! -path "**/.history/*" \
! -path "**/__pycache__/*" \
! -path "./test/*" \
! -path "./dev/config/*" \
! -path "./bin/*" \
! -path "./.circleci/*" \
! -path "./.git/*" \
! -name LICENSE \
! -name "*requirements.txt" \
! -name "go.*" \
! -name "*.md" \
! -name ".*" \
! -name "*.bin" \
! -name "Dockerfile" \
-exec grep -L "Copyright 2021 Cortex Labs, Inc" {} \;)
if [[ $output ]]; then
  echo "File(s) are missing Cortex license:"
  echo "$output"
  exit 1
fi

if [ "$is_release_branch" = "true" ]; then
  # Check for occurrences of "master" which should be changed to the version number
  output=$(cd "$ROOT" && find . -type f \
  ! -path "./build/lint.sh" \
  ! -path "./vendor/*" \
  ! -path "**/.vscode/*" \
  ! -path "**/.idea/*" \
  ! -path "**/.history/*" \
  ! -path "**/__pycache__/*" \
  ! -path "./dev/config/*" \
  ! -path "./bin/*" \
  ! -path "./.git/*" \
  ! -name ".*" \
  ! -name "*.bin" \
  -exec grep -R -A 5 -e "CORTEX_VERSION" {} \;)
  output=$(echo "$output" | grep -e "master" || true)
  if [[ $output ]]; then
    echo 'occurrences of "master" which should be changed to the version number:'
    echo "$output"
    exit 1
  fi
fi

# Check for trailing whitespace
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "**/.idea/*" \
! -path "**/.history/*" \
! -path "**/.vscode/*" \
! -path "**/__pycache__/*" \
! -path "./dev/config/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -name ".*" \
! -name "*.bin" \
! -name "*.wav" \
-exec egrep -l " +$" {} \;)
if [[ $output ]]; then
  echo "File(s) have lines with trailing whitespace:"
  echo "$output"
  exit 1
fi

# Check for missing new line at end of file
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "**/.idea/*" \
! -path "**/.history/*" \
! -path "**/.vscode/*" \
! -path "**/__pycache__/*" \
! -path "./dev/config/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -name ".*" \
! -name "*.bin" \
! -name "*.wav" \
-print0 | \
xargs -0 -L1 bash -c 'test "$(tail -c 1 "$0")" && echo "No new line at end of $0"' || true)
if [[ $output ]]; then
  echo "$output"
  exit 1
fi

# Check for multiple new lines at end of file
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "**/.vscode/*" \
! -path "**/.idea/*" \
! -path "**/.history/*" \
! -path "**/__pycache__/*" \
! -path "./dev/config/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -name ".*" \
! -name "*.bin" \
! -name "*.wav" \
-print0 | \
xargs -0 -L1 bash -c 'test "$(tail -c 2 "$0")" || [ ! -s "$0" ] || echo "Multiple new lines at end of $0"' || true)
if [[ $output ]]; then
  echo "$output"
  exit 1
fi

# Check for new line(s) at beginning of file
output=$(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "**/.idea/*" \
! -path "**/.history/*" \
! -path "**/.vscode/*" \
! -path "**/__pycache__/*" \
! -path "./dev/config/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -name ".*" \
! -name "*.bin" \
! -name "*.wav" \
-print0 | \
xargs -0 -L1 bash -c 'test "$(head -c 1 "$0")" || [ ! -s "$0" ] || echo "New line at beginning of $0"' || true)
if [[ $output ]]; then
  echo "$output"
  exit 1
fi

# Check docs links
output=$(python3 $ROOT/dev/find_missing_docs_links.py)
if [[ $output ]]; then
  echo "docs file(s) have broken links:"
  echo "$output"
  exit 1
fi
