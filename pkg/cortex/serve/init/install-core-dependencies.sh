#!/usr/bin/env bash

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

function module_exists() {
    module_name=$1
    python -c "import importlib.util, sys; loader = importlib.util.find_spec('$module_name'); sys.exit(1) if loader is None else sys.exit(0)"
}

if ! module_exists "cortex_internal"; then
    pip install --no-cache-dir -U \
        -r /src/cortex/serve/serve.requirements.txt \
        /src/cortex/serve/
fi

if [ "${CORTEX_IMAGE_TYPE}" = "tensorflow-handler" ]; then
    if ! module_exists "tensorflow" || ! module_exists "tensorflow_serving"; then
        pip install --no-cache-dir -U \
            tensorflow-cpu==2.3.0 \
            tensorflow-serving-api==2.3.0
    fi
fi
