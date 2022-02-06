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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"
ENVTEST_ASSETS_DIR=${ROOT}/testbin

cluster_env="undefined"
create_cluster="no"
positional_args=()
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -e|--cluster-env)
    cluster_env="$2"
    shift
    ;;
    -c|--create-cluster)
    create_cluster="yes"
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
    -e=*|--cluster-env=*)
    cluster_env="${i#*=}"
    shift
    ;;
    -c|--create-cluster)
    create_cluster="yes"
    shift
    ;;
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

cmd=${1:-""}
sub_cmd=${2:-""}

function run_go_tests() {
  (
    cd $ROOT
    mkdir -p ${ENVTEST_ASSETS_DIR}
	  test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	  source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools ${ENVTEST_ASSETS_DIR}; setup_envtest_env ${ENVTEST_ASSETS_DIR}
    go test -race ./... && echo "go tests passed"
  )
}

function run_e2e_tests() {
  if [ "$create_cluster" = "yes" ]; then
    pytest $ROOT/test/e2e/tests --config "$sub_cmd"
  else
    pytest $ROOT/test/e2e/tests --env "$cluster_env"
  fi
}

if [ "$cmd" = "go" ]; then
  run_go_tests
elif [ "$cmd" = "e2e" ]; then
  run_e2e_tests
fi
