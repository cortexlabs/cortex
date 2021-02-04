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
    python -c "import importlib, sys; loader = importlib.util.find_spec('$module_name'); sys.exit(1) if loader is None else sys.exit(0)"
}

temp_freeze=$(mktemp)
pip freeze > "${temp_freeze}"

if ! grep -q "cortex-internal" "${temp_freeze}"; then
    pip install --no-cache-dir -U \
        -r /src/cortex/serve/serve.requirements.txt \
        /src/cortex/serve/
fi

rm "${temp_freeze}"

if [ "${CORTEX_IMAGE_TYPE}" = "tensorflow-predictor" ]; then
    if ! module_exists "tensorflow" || ! module_exists "tensorflow_serving"; then
        pip install --no-cache-dir -U \
            tensorflow-cpu==2.3.0 \
            tensorflow-serving-api==2.3.0
    fi
elif [ "${CORTEX_IMAGE_TYPE}" = "onnx-predictor-cpu" ] && ! module_exists "onnxruntime"; then
    pip install --no-cache-dir -U \
        onnxruntime==1.4.0
elif [ "${CORTEX_IMAGE_TYPE}" = "onnx-predictor-gpu" ] && ! module_exists "onnxruntime"; then
    pip install --no-cache-dir -U \
        onnxruntime-gpu==1.4.0
fi
