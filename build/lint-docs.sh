#!/bin/bash

# Copyright 2022 Cortex Labs, Inc.
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

# This is a subset of lint.sh, and is only meant to be run on master

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

# Check docs links
output=$(python3 $ROOT/dev/find_missing_docs_links.py)
if [[ $output ]]; then
  echo "docs file(s) have broken links:"
  echo "$output"
  exit 1
fi

# Check for trailing whitespace
output=$(cd "$ROOT/docs" && find . -type f \
-exec egrep -l " +$" {} \;)
if [[ $output ]]; then
  echo "File(s) have lines with trailing whitespace:"
  echo "$output"
  exit 1
fi

# Check for missing new line at end of file
output=$(cd "$ROOT/docs" && find . -type f \
-print0 | \
xargs -0 -L1 bash -c 'test "$(tail -c 1 "$0")" && echo "No new line at end of $0"' || true)
if [[ $output ]]; then
  echo "$output"
  exit 1
fi

# Check for multiple new lines at end of file
output=$(cd "$ROOT/docs" && find . -type f \
-print0 | \
xargs -0 -L1 bash -c 'test "$(tail -c 2 "$0")" || echo "Multiple new lines at end of $0"' || true)
if [[ $output ]]; then
  echo "$output"
  exit 1
fi

# Check for new line(s) at beginning of file
output=$(cd "$ROOT/docs" && find . -type f \
-print0 | \
xargs -0 -L1 bash -c 'test "$(head -c 1 "$0")" || echo "New line at beginning of $0"' || true)
if [[ $output ]]; then
  echo "$output"
  exit 1
fi
