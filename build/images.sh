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

# images to build/push for development and CI commands
# each image should appear exactly once on this page

set -euo pipefail

dev_images=(
  "manager"
  "proxy"
  "async-gateway"
  "enqueuer"
  "dequeuer"
  "autoscaler"
  "activator"
)

non_dev_images=(
  "cluster-autoscaler"
  "operator"
  "controller-manager"
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
  "metrics-server"
  "nvidia-device-plugin"
  "neuron-device-plugin"
  "neuron-scheduler"
  "kubexit"
)

# for linux/amd64 and linux/arm64
multi_arch_images=(
  "proxy"
  "async-gateway"
  "enqueuer"
  "dequeuer"
  "fluent-bit"
  "prometheus-node-exporter"
  "kube-rbac-proxy"
  "kubexit"
)

all_images=(
  "${dev_images[@]}"
  "${non_dev_images[@]}"
)
