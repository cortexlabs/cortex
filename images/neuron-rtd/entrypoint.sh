#!/usr/bin/env bash

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

start_cmd="neuron-rtd -g unix:/sock/neuron.sock --log-console"
if [ -f "/mnt/kubexit" ]; then
  start_cmd="/mnt/kubexit ${start_cmd}"
fi

eval "${start_cmd}"
