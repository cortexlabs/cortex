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

set -e

# empty model config list
mkdir /etc/tfs
echo "model_config_list {}" > /etc/tfs/model_config_server.conf

# configure batching if specified
if [[ -n ${TF_MAX_BATCH_SIZE} && -n ${TF_BATCH_TIMEOUT_MICROS} ]]; then
    echo "max_batch_size { value: ${TF_MAX_BATCH_SIZE} }" > /etc/tfs/batch_config.conf
    echo "batch_timeout_micros { value: ${TF_BATCH_TIMEOUT_MICROS} }" >> /etc/tfs/batch_config.conf
    echo "num_batch_threads { value: ${TF_NUM_BATCHED_THREADS} }" >> /etc/tfs/batch_config.conf
fi

# launch TFS
tensorflow_model_server "$@"
