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

kill $(pgrep -f rerun) >/dev/null 2>&1 || true

eval $(python3 $ROOT/manager/cluster_config_env.py "$ROOT/dev/config/cluster.yaml")

python3 $ROOT/manager/update_cli_config.py "$HOME/.cortex/cli.yaml" "default" "http://localhost:8888" "$CORTEX_AWS_ACCESS_KEY_ID" "$CORTEX_AWS_SECRET_ACCESS_KEY"

cp -r $ROOT/dev/config/cluster.yaml ~/.cortex/cluster-dev.yaml

if grep -qiP '^telemetry:\s*false\s*$' ~/.cortex/cli.yaml; then
  echo "telemetry: false" >> ~/.cortex/cluster-dev.yaml
fi

export CORTEX_OPERATOR_IN_CLUSTER=false
export CORTEX_CLUSTER_CONFIG_PATH=~/.cortex/cluster-dev.yaml

rerun -watch $ROOT/pkg $ROOT/cli $ROOT/dev/config -ignore $ROOT/vendor $ROOT/bin -run sh -c \
"go build -o $ROOT/bin/operator $ROOT/pkg/operator && go build -installsuffix cgo -o $ROOT/bin/cortex $ROOT/cli && $ROOT/bin/operator"

# go run -race $ROOT/pkg/operator/main.go  # Check for race conditions. Doesn't seem to catch them all?
