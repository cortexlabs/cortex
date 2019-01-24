#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

source $ROOT/dev/config/cortex.sh

export CONST_OPERATOR_TRANSFORMERS_DIR=$ROOT/pkg/transformers
export CONST_OPERATOR_AGGREGATORS_DIR=$ROOT/pkg/aggregators
export CONST_OPERATOR_IN_CLUSTER=false

if ! command -v rerun > /dev/null; then
  GO111MODULE=off go get -u -v github.com/VojtechVitek/rerun/cmd/rerun
fi
rerun -watch $ROOT/pkg -ignore $ROOT/vendor $ROOT/bin -run sh -c "go run $ROOT/pkg/operator/operator.go"
# go run -race $ROOT/pkg/operator/operator.go
