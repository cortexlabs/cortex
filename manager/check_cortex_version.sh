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

set -e

CORTEX_VERSION=master

if [ "$CORTEX_VERSION" != "$CORTEX_CLI_VERSION" ]; then
  echo "error: your CLI version ($CORTEX_CLI_VERSION) doesn't match your Cortex manager image version ($CORTEX_VERSION); please update your CLI (pip install cortex==$CORTEX_VERSION) to match the version of your Cortex manager image"
  exit 1
fi
