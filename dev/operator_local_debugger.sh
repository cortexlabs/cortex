#!/bin/bash

# Copyright 2020 Cortex Labs, Inc.
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

arg1=${1:-""}

provider=""
if [ "$arg1" = "--aws" ]; then
  provider="aws"
elif [ "$arg1" = "--gcp" ]; then
  provider="gcp"
else
  echo "provider must be set: either pass in the --aws or the --gcp flag"
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"
DEBUG_CMD="dlv --listen=:2345 --headless=true --api-version=2 debug $ROOT/pkg/operator --output ${ROOT}/bin/__debug_bin"

kill $(pgrep -f "${DEBUG_CMD}") >/dev/null 2>&1 || true
kill $(pgrep -f __debug_bin) >/dev/null 2>&1 || true

eval $(python3 $ROOT/manager/cluster_config_env.py "$ROOT/dev/config/cluster-${provider}.yaml")

if [ "$provider" = "aws" ]; then
  export CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY="$CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY_AWS"
  export CLUSTER_AWS_ACCESS_KEY_ID="${CLUSTER_AWS_ACCESS_KEY_ID:-$AWS_ACCESS_KEY_ID}"
  export CLUSTER_AWS_SECRET_ACCESS_KEY="${CLUSTER_AWS_SECRET_ACCESS_KEY:-$AWS_SECRET_ACCESS_KEY}"
else
  export CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY="$CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY_GCP"
fi

python3 $ROOT/dev/update_cli_config.py "$HOME/.cortex/cli.yaml" "${CORTEX_CLUSTER_NAME}-${provider}" "$provider" "http://localhost:8888"

cp -r $ROOT/dev/config/cluster-${provider}.yaml ~/.cortex/cluster-dev.yaml

if grep -qiP '^telemetry:\s*false\s*$' ~/.cortex/cli.yaml; then
  echo "telemetry: false" >> ~/.cortex/cluster-dev.yaml
fi

export CORTEX_OPERATOR_IN_CLUSTER=false
export CORTEX_CLUSTER_CONFIG_PATH=~/.cortex/cluster-dev.yaml

mkdir -p $ROOT/bin
echo 'starting local operator in debug mode...' && eval "${DEBUG_CMD}"
