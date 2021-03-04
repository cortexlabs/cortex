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


err=0
trap 'err=1' ERR

function substitute_env_vars() {
    file_to_run_substitution=$1
    python -c "from cortex_internal.lib import util; import os; util.expand_environment_vars_on_file('$file_to_run_substitution')"
}

substitute_env_vars $CORTEX_LOG_CONFIG_FILE
pytest lib/test

test $err = 0
