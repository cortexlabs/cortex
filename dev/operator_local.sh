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

source $ROOT/dev/config/cortex.sh

export CORTEX_OPERATOR_TRANSFORMERS_DIR=$ROOT/pkg/transformers
export CORTEX_OPERATOR_AGGREGATORS_DIR=$ROOT/pkg/aggregators
export CORTEX_OPERATOR_IN_CLUSTER=false

is_local=${1:-""}

if [ "$is_local" == "local" ]; then
    export CORTEX_CLOUD_PROVIDER_TYPE="local"
    export CORTEX_OPERATOR_LOCAL_MOUNT="$HOME/.cortex"
    export CORTEX_BUCKET="/mount"
    export CORTEX_OPERATOR_HOST_IP="192.168.99.115"
fi

rerun -watch $ROOT/pkg $ROOT/cli -ignore $ROOT/vendor $ROOT/bin -run sh -c \
"go build -o $ROOT/bin/operator $ROOT/pkg/operator && go build -installsuffix cgo -o $ROOT/bin/cortex $ROOT/cli && $ROOT/bin/operator"
# go run -race $ROOT/pkg/operator/operator.go
