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


# USAGE: ./dev/python_version_test.sh <python version> <env_name>
# e.g.: ./dev/python_version_test.sh 3.6.9 aws

set -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

# create a new conda environment based on the supplied python version
conda create -n env -y
CONDA_BASE=$(conda info --base)
source $CONDA_BASE/etc/profile.d/conda.sh
conda activate env
conda config --append channels conda-forge
conda install python=$1 -y

pip install requests

export CORTEX_CLI_PATH=$ROOT/bin/cortex

# install cortex
cd $ROOT/pkg/cortex/client
pip install -e .

# run script.py
python $ROOT/dev/deploy_test.py $2

# clean up conda
conda deactivate
conda env remove -n env
rm -rf $ROOT/pkg/cortex/client/cortex.egg-info
