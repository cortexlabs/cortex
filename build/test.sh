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


set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

provider="undefined"
cluster_env="undefined"
create_cluster="no"
positional_args=()
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -p|--provider)
    provider="$2"
    shift
    ;;
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
    -p=*|--provider=*)
    provider="${i#*=}"
    shift
    ;;
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
  (cd $ROOT && go test ./... && echo "go tests passed")
}

function run_python_tests() {
  docker build $ROOT -f $ROOT/images/test/Dockerfile -t cortexlabs/test
  docker run cortexlabs/test
}

function run_e2e_tests() {
  if [ "$provider" = "aws" ]; then
    if [ "$create_cluster" = "yes" ]; then
      pytest $ROOT/test/e2e/tests -k aws --aws-config "$sub_cmd"
    else
      pytest $ROOT/test/e2e/tests -k aws --aws-env "$cluster_env"
    fi
  elif [ "$provider" = "gcp" ]; then
    if [ "$create_cluster" = "yes" ]; then
      pytest $ROOT/test/e2e/tests -k gcp --gcp-config "$sub_cmd"
    else
      pytest $ROOT/test/e2e/tests -k gcp --gcp-env "$cluster_env"
    fi
  fi
}

if [ "$cmd" = "go" ]; then
  run_go_tests
elif [ "$cmd" = "python" ]; then
  run_python_tests
elif [ "$cmd" = "e2e" ]; then
  run_e2e_tests
else
  run_go_tests
  run_python_tests
fi
