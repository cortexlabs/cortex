#!/usr/bin/env bash

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

# This is a subset of lint.sh, and is only meant to be run on master

CLUSTER_CONFIG=$1

port_forward_cmd="kubectl port-forward -n prometheus prometheus-prometheus-0 9090"
kill $(pgrep -f "${port_forward_cmd}") >/dev/null 2>&1 || true

echo "Port-forwarding Prometheus to localhost:9090"
eval "${port_forward_cmd}" >/dev/null 2>&1 &

CORTEX_DISABLE_JSON_LOGGING="true" \
CORTEX_PROMETHEUS_URL="http://localhost:9090" \
go run ./main.go -config "${CLUSTER_CONFIG}"
