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

cd /mnt/project

# If the container restarted, ensure that it is not perceived as ready
rm -rf /mnt/api_readiness.txt

# Allow for the liveness check to pass until the API is running
echo "9999999999" > /mnt/api_liveness.txt

export PYTHONPATH=$PYTHONPATH:$PYTHON_PATH

sysctl -w net.core.somaxconn=$CORTEX_SO_MAX_CONN >/dev/null
sysctl -w net.ipv4.ip_local_port_range="15000 64000" >/dev/null
sysctl -w net.ipv4.tcp_fin_timeout=30 >/dev/null

# execute script if present in project's directory
if [ -f "/mnt/project/script.sh" ]; then
    chmod +x /mnt/project/script.sh
    /mnt/project/script.sh
fi

# overwrite conda config file if it exists
if [ -f "/mnt/project/.condarc" ]; then
    cp /mnt/project/.condarc /root/.condarc
fi

# either install from environment.yaml or from conda-packages.txt
if [ -f "/mnt/project/environment.yaml" ]; then
    conda env update --name env --file /mnt/project/environment.yaml
elif [ -f "/mnt/project/conda-packages.txt" ]; then
    conda install --file /mnt/project/conda-packages.txt
fi

# install pip packages
if [ -f "/mnt/project/requirements.txt" ]; then
    pip --no-cache-dir install -r /mnt/project/requirements.txt
fi

# Ensure predictor print() statements are always flushed
export PYTHONUNBUFFERED=TRUE

mkdir -p /mnt/requests

/opt/conda/envs/env/bin/python /src/cortex/serve/start_uvicorn.py
