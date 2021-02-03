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

temp_freeze=$(mktemp)
new_reqs=$(mktemp)
pip freeze > "${temp_freeze}"

while IFS= read -r dependency; do
    if ! grep -Fxq "${dependency}" "${temp_freeze}"; then
        echo "${dependency}" >> "${new_reqs}"
    fi
done < /src/cortex/serve/serve.requirements.txt

if [ $(cat "${new_reqs}" | wc -l ) -ne 0 ]; then
    if grep -q "cortex-internal" "${temp_freeze}"; then
        pip install --no-cache-dir -U \
            -r "${new_reqs}"
    else
        pip install --no-cache-dir -U \
            -r "${new_reqs}" \
            /src/cortex/serve/
    fi
fi

rm "${new_reqs}"
new_reqs=$(mktemp)

if [ -f "/src/cortex/serve/image.requirements.txt" ]; then
    while IFS= read -r dependency; do
        if ! grep -Fxq "${dependency}" "${temp_freeze}"; then
            echo "${dependency}" >> "${new_reqs}"
        fi
    done < /src/cortex/serve/image.requirements.txt

    if [ $(cat "${new_reqs}" | wc -l ) -ne 0 ]; then
        pip install --no-cache-dir -U \
            -r "${new_reqs}"
    fi
fi

rm "${new_reqs}"
