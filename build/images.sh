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

api_images_cluster=(
  "python-predictor-cpu"
  "python-predictor-gpu"
  "tensorflow-predictor"
  "onnx-predictor-cpu"
  "onnx-predictor-gpu"
)
api_images_aws=(
  # includes api_images_cluster
  "python-predictor-inf"
)
api_images_gcp=(
  # includes api_images_cluster
)

dev_images_cluster=(
  "downloader"
  "manager"
  "request-monitor"
)
dev_images_aws=(
  # includes dev_images_cluster
)
dev_images_gcp=(
  # includes dev_images_cluster
)

non_dev_images_cluster=(
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
  "grafana"
  "event-exporter"
)
non_dev_images_aws=(
  # includes non_dev_images_cluster
  "tensorflow-serving-inf"
  "metrics-server"
  "inferentia"
  "neuron-rtd"
  "nvidia"
)
non_dev_images_gcp=(
  # includes non_dev_images_cluster
  "google-pause"
)

all_images=(
  "${api_images_cluster[@]}"
  "${api_images_aws[@]}"
  "${api_images_gcp[@]}"
  "${dev_images_cluster[@]}"
  "${dev_images_aws[@]}"
  "${dev_images_gcp[@]}"
  "${non_dev_images_cluster[@]}"
  "${non_dev_images_aws[@]}"
  "${non_dev_images_gcp[@]}"
)

aws_images=(
  "${api_images_cluster[@]}"
  "${api_images_aws[@]}"
  "${dev_images_cluster[@]}"
  "${dev_images_aws[@]}"
  "${non_dev_images_cluster[@]}"
  "${non_dev_images_aws[@]}"
)

gcp_images=(
  "${api_images_cluster[@]}"
  "${api_images_gcp[@]}"
  "${dev_images_cluster[@]}"
  "${dev_images_gcp[@]}"
  "${non_dev_images_cluster[@]}"
  "${non_dev_images_gcp[@]}"
)
