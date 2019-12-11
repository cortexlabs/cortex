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

git_branch="${CIRCLE_BRANCH:-""}"
if [ "$git_branch" = "" ]; then
  git_branch=$(cd "$ROOT" && git rev-parse --abbrev-ref HEAD)
fi

if ! echo "$git_branch" | grep -Eq ^[0-9]+.[0-9]+$; then
  echo "error: this is meant to be run on release branches"
  exit 1
fi

if [[ "$OSTYPE" == "darwin"* ]]; then
  cd "$ROOT" && find . -type f \
  ! -path "./build/lint.sh" \
  ! -path "./dev/update_version_comments.sh" \
  ! -path "./vendor/*" \
  ! -path "./bin/*" \
  ! -path "./.git/*" \
  ! -name ".*" \
  -print0 | \
  xargs -0 sed -i '' -e '/.*WARNING: you are on the master branch, please refer to the docs on the branch that matches.*$/N; /.*WARNING: you are on the master branch, please refer to the docs on the branch that matches.*/d'
else
  cd "$ROOT" && find . -type f \
  ! -path "./build/lint.sh" \
  ! -path "./dev/update_version_comments.sh" \
  ! -path "./vendor/*" \
  ! -path "./bin/*" \
  ! -path "./.git/*" \
  ! -name ".*" \
  -print0 | \
  xargs -0 sed -i '/.*WARNING: you are on the master branch, please refer to the docs on the branch that matches.*/,+1 d'
fi

if [[ "$OSTYPE" == "darwin"* ]]; then
  cd "$ROOT" && find . -type f \
  ! -path "./build/lint.sh" \
  ! -path "./dev/update_version_comments.sh" \
  ! -path "./vendor/*" \
  ! -path "./bin/*" \
  ! -path "./.git/*" \
  ! -name ".*" \
  -print0 | \
  xargs -0 sed -i '' -e "s/WARNING: you are on the master branch, please refer to the examples on the branch that matches/this is an example for cortex release ${git_branch} and may not deploy correctly on other releases of cortex/"
else
  cd "$ROOT" && find . -type f \
  ! -path "./build/lint.sh" \
  ! -path "./dev/update_version_comments.sh" \
  ! -path "./vendor/*" \
  ! -path "./bin/*" \
  ! -path "./.git/*" \
  ! -name ".*" \
  -print0 | \
  xargs -0 sed -i "s/WARNING: you are on the master branch, please refer to the examples on the branch that matches/this is an example for cortex release ${git_branch} and may not deploy correctly on other releases of cortex/"
fi

echo "done"
