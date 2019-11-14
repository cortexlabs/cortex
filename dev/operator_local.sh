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

export CORTEX_OPERATOR_IN_CLUSTER=false
export CORTEX_CLUSTER_CONFIG_PATH=$ROOT/dev/config/cluster.yaml
export CORTEX_INTERNAL_CLUSTER_CONFIG_PATH=$HOME/.cortex/cluster_internal.yaml

pip3 install -r $ROOT/manager/requirements.txt

python3 $ROOT/manager/instance_metadata.py $CORTEX_CLUSTER_CONFIG_PATH $CORTEX_INTERNAL_CLUSTER_CONFIG_PATH

kill $(pgrep -f rerun) >/dev/null 2>&1 || true
updated_cli_config=$(cat $HOME/.cortex/default.json | jq '.cortex_url = "http://localhost:8888"') && echo $updated_cli_config > $HOME/.cortex/default.json
rerun -watch $ROOT/pkg $ROOT/cli $ROOT/dev/config -ignore $ROOT/vendor $ROOT/bin -run sh -c \
"go build -o $ROOT/bin/operator $ROOT/pkg/operator && go build -installsuffix cgo -o $ROOT/bin/cortex $ROOT/cli && $ROOT/bin/operator"

# go run -race $ROOT/pkg/operator/operator.go  # Check for race conditions. Doesn't seem to catch them all?
