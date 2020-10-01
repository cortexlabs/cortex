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

# images to build when running a registry command
registry_all_images=(
"tensorflow-serving-cpu"
"tensorflow-serving-gpu"
"tensorflow-serving-inf"
"cluster-autoscaler"
"metrics-server"
"inferentia"
"neuron-rtd"
"nvidia"
"fluentd"
"statsd"
"istio-proxy"
"istio-pilot"
"istio-citadel"
"istio-galley"
)

# images to build when running a registry command
registry_dev_images=( "manager" "downloader" )

# images to build when running a registry command
registry_base_images=(
"python-predictor-cpu --include-slim"
"python-predictor-gpu --include-slim"
"python-predictor-inf --include-slim"
"tensorflow-predictor --include-slim"
"onnx-predictor-cpu --include-slim"
"onnx-predictor-gpu --include-slim"
)

# images to build and push for the CI
ci_images=( "${registry_all_images[@]}" "${registry_dev_images[@]}" "${registry_base_images[@]}" )
