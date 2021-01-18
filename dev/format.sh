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

if ! command -v gofmt >/dev/null 2>&1; then
  echo "gofmt must be installed"
  exit 1
fi

if ! command -v black >/dev/null 2>&1; then
  echo "black must be installed"
  exit 1
fi

gofmt -s -w "$ROOT"/cli "${ROOT}"/pkg

black --quiet --line-length=100 --exclude .idea/ "$ROOT"

# Trim trailing whitespace
if [[ "$OSTYPE" == "darwin"* ]]; then
  output=$(cd "$ROOT" && find . -type f \
  ! -path "./vendor/*" \
  ! -path "./bin/*" \
  ! -path "./.git/*" \
  ! -path "./.idea/*" \
  ! -name ".*" \
  -print0 | \
  xargs -0 sed -i '' -e's/[[:space:]]*$//')
else
  output=$(cd "$ROOT" && find . -type f \
  ! -path "./vendor/*" \
  ! -path "./bin/*" \
  ! -path "./.git/*" \
  ! -path "./.idea/*" \
  ! -name ".*" \
  -print0 | \
  xargs -0 sed -i 's/[[:space:]]*$//')
fi

# Add new line to end of file
(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -path "./.idea/*" \
! -name ".*" \
-print0 | \
xargs -0 -L1 bash -c 'test "$(tail -c 1 "$0")" && echo "" >> "$0"' || true)

# Remove repeated new lines at end of file
(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -path "./.idea/*" \
! -name ".*" \
-print0 | \
xargs -0 -L1 bash -c 'test "$(tail -c 2 "$0")" || [ ! -s "$0" ] || (trimmed=$(printf "%s" "$(< $0)") && echo "$trimmed" > "$0")' || true)

# Remove new lines at beginning of file
(cd "$ROOT" && find . -type f \
! -path "./vendor/*" \
! -path "./bin/*" \
! -path "./.git/*" \
! -path "./.idea/*" \
! -name ".*" \
-print0 | \
xargs -0 -L1 bash -c 'test "$(head -c 1 "$0")" || [ ! -s "$0" ] || (trimmed=$(sed '"'"'/./,$!d'"'"' "$0") && echo "$trimmed" > "$0")' || true)
