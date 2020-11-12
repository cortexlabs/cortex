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

# images to build/push for development and CI commands

set -euo pipefail

user_facing_images=(
  "python-predictor-cpu"
  # "python-predictor-gpu"
  # "python-predictor-inf"
  "tensorflow-predictor"
  "onnx-predictor-cpu"
  # "onnx-predictor-gpu"
)

user_facing_slim_images=(
  "python-predictor-cpu-slim"
  "python-predictor-gpu-slim"
  "python-predictor-inf-slim"
  "tensorflow-predictor-slim"
  "onnx-predictor-cpu-slim"
  "onnx-predictor-gpu-slim"
)

dev_images=(
  "manager"
  "request-monitor"
  "downloader"
)

non_dev_images=(
  "operator"
  "tensorflow-serving-cpu"
  "tensorflow-serving-gpu"
  # "tensorflow-serving-inf"
  "cluster-autoscaler"
  "metrics-server"
  # "inferentia"
  # "neuron-rtd"
  "nvidia"
  "fluentd"
  "statsd"
  "istio-proxy"
  "istio-pilot"
)

all_images=( "${user_facing_images[@]}" "${user_facing_slim_images[@]}" "${dev_images[@]}" "${non_dev_images[@]}" )
