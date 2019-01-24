#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

function run_go_tests() {
  (cd $ROOT && GO111MODULE=on go test ./... && echo -e "\ngo vet..." && GO111MODULE=on go vet ./... && echo "go tests passed")
}

function run_python_tests() {
  docker build $ROOT -f $ROOT/images/test/Dockerfile -t cortexlabs/test
  docker run cortexlabs/test
}

CMD=${1:-""}

if [ "$CMD" = "go" ]; then
  run_go_tests
elif [ "$CMD" = "python" ]; then
  run_python_tests
else
  run_go_tests
  run_python_tests
fi
