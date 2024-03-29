#!/bin/bash

# Copyright 2022 Cortex Labs, Inc.
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

operator_only="false"
debug="false"
positional_args=()
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --operator-only)
    operator_only="true"
    shift
    ;;
    --debug)
    debug="true"
    shift
    ;;
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done
set -- "${positional_args[@]}"
positional_args=()
for i in "$@"; do
  case $i in
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done
set -- "${positional_args[@]}"
for arg in "$@"; do
  if [[ "$arg" == -* ]]; then
    echo "unknown flag: $arg"
    exit 1
  fi
done

if [ "$operator_only" = "true" ] && [ "$debug" = "true" ]; then
  echo "error: --operator-only and --debug cannot both be set"
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

eval $(python3 $ROOT/manager/cluster_config_env.py "$ROOT/dev/config/cluster.yaml")

export CORTEX_DEV_DEFAULT_IMAGE_REGISTRY="$CORTEX_DEV_DEFAULT_IMAGE_REGISTRY"

python3 $ROOT/dev/update_cli_config.py "$HOME/.cortex/cli.yaml" "${CORTEX_CLUSTER_NAME}" "http://localhost:8888"

cp -r $ROOT/dev/config/cluster.yaml ~/.cortex/cluster-dev.yaml

if grep -qiP '^telemetry:\s*false\s*$' ~/.cortex/cli.yaml; then
  echo "telemetry: false" >> ~/.cortex/cluster-dev.yaml
fi

export CORTEX_OPERATOR_IN_CLUSTER=false
export CORTEX_CLUSTER_CONFIG_PATH=~/.cortex/cluster-dev.yaml
export CORTEX_DISABLE_JSON_LOGGING=true
export CORTEX_LOG_LEVEL=debug
export CORTEX_PROMETHEUS_URL="http://localhost:9090"

portForwardCMD="kubectl port-forward -n default prometheus-prometheus-0 9090"
kill $(pgrep -f "${portForwardCMD}") >/dev/null 2>&1 || true

echo "Port-forwarding Prometheus to localhost:9090"
eval "${portForwardCMD}" >/dev/null 2>&1 &

mkdir -p $ROOT/bin

if [ "$operator_only" = "true" ]; then
  kill $(pgrep -f rerun) >/dev/null 2>&1 || true
  rerun -watch $ROOT/pkg $ROOT/dev/config $ROOT/cmd/operator -run sh -c \
  "clear && echo 'building operator...' && go build -o $ROOT/bin/operator $ROOT/cmd/operator && echo 'starting local operator...' && $ROOT/bin/operator"
elif [ "$debug" = "true" ]; then
  DEBUG_CMD="dlv --listen=:2345 --headless=true --api-version=2 debug $ROOT/cmd/operator --output ${ROOT}/bin/__debug_bin"
  kill $(pgrep -f "${DEBUG_CMD}") >/dev/null 2>&1 || true
  kill $(pgrep -f __debug_bin) >/dev/null 2>&1 || true
  echo 'starting local operator in debug mode...' && eval "${DEBUG_CMD}"
else
  kill $(pgrep -f rerun) >/dev/null 2>&1 || true
  rerun -watch $ROOT/pkg $ROOT/cli $ROOT/dev/config $ROOT/cmd/operator -run sh -c \
  "clear && echo 'building cli...' && go build -o $ROOT/bin/cortex $ROOT/cli && echo 'building operator...' && go build -o $ROOT/bin/operator $ROOT/cmd/operator && echo 'starting local operator...' && $ROOT/bin/operator"
fi

# go run -race $ROOT/cmd/operator # Check for race conditions. Doesn't seem to catch them all?
