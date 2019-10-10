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

export CORTEX_OPERATOR_IN_CLUSTER=false

pip3 install -r $ROOT/manager/requirements.txt

export CORTEX_NODE_CPU=$(python3 $ROOT/manager/instance_metadata.py --region=$CORTEX_REGION --instance-type=$CORTEX_NODE_TYPE --cache-dir="$HOME/.cortex/ec2-metadata.json" --feature="cpu")
export CORTEX_NODE_MEM=$(python3 $ROOT/manager/instance_metadata.py --region=$CORTEX_REGION --instance-type=$CORTEX_NODE_TYPE --cache-dir="$HOME/.cortex/ec2-metadata.json" --feature="mem")
export CORTEX_NODE_GPU=$(python3 $ROOT/manager/instance_metadata.py --region=$CORTEX_REGION --instance-type=$CORTEX_NODE_TYPE --cache-dir="$HOME/.cortex/ec2-metadata.json" --feature="gpu")

kill $(pgrep -f rerun) >/dev/null 2>&1 || true
updated_config=$(cat $HOME/.cortex/default.json | jq '.cortex_url = "http://localhost:8888"') && echo $updated_config > $HOME/.cortex/default.json
rerun -watch $ROOT/pkg $ROOT/cli -ignore $ROOT/vendor $ROOT/bin -run sh -c \
"go build -o $ROOT/bin/operator $ROOT/pkg/operator && go build -installsuffix cgo -o $ROOT/bin/cortex $ROOT/cli && $ROOT/bin/operator"

# go run -race $ROOT/pkg/operator/operator.go  # Check for race conditions. Doesn't seem to catch them all?
