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

set -eo pipefail

api_images=(
  "async-text-generator-cpu"
  "async-text-generator-gpu"
  "batch-image-classifier-alexnet-cpu"
  "batch-image-classifier-alexnet-gpu"
  "batch-sum-cpu"
  "realtime-image-classifier-resnet50-cpu"
  "realtime-image-classifier-resnet50-gpu"
  "realtime-prime-generator-cpu"
  "realtime-sleep-cpu"
  "realtime-text-generator-cpu"
  "realtime-text-generator-gpu"
  "task-iris-classifier-trainer-cpu"
  "trafficsplitter-hello-world-cpu"
)
