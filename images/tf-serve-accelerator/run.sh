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

set -e

export base_port=${1#*=}
export model_path=${2#*=}
export model_name=${3#*=}

for i in $(seq 1 $TF_WORKERS); do
    echo -e "\n\n" >> /tmp/supervisord.conf
    worker=$i port=$((base_port+i-1)) envsubst < /tmp/template.conf >> /tmp/supervisord.conf
done

cp /tmp/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
/usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf