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

# images to build/push for development and CI commands
# each image should appear exactly once on this page

set -euo pipefail

api_images=(
  "python-predictor-cpu"
  "python-predictor-gpu"
  "tensorflow-predictor"
  "python-predictor-inf"
)

dev_images=(
  "downloader"
  "manager"
  "request-monitor"
  "async-gateway"
)

non_dev_images=(
  "tensorflow-serving-cpu"
  "tensorflow-serving-gpu"
  "cluster-autoscaler"
  "operator"
  "istio-proxy"
  "istio-pilot"
  "fluent-bit"
  "prometheus"
  "prometheus-config-reloader"
  "prometheus-operator"
  "prometheus-statsd-exporter"
  "prometheus-dcgm-exporter"
  "prometheus-kube-state-metrics"
  "prometheus-node-exporter"
  "kube-rbac-proxy"
  "grafana"
  "event-exporter"
  "tensorflow-serving-inf"
  "metrics-server"
  "inferentia"
  "neuron-rtd"
  "nvidia"
)

all_images=(
  "${api_images[@]}"
  "${dev_images[@]}"
  "${non_dev_images[@]}"
)
