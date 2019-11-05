# Copyright 2019 Cortex Labs, Inc.
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

import sys
import yaml

cluster_conifg_path = sys.argv[1]

with open(cluster_conifg_path, "r") as cluster_conifg_file:
    cluster_conifg = yaml.safe_load(cluster_conifg_file)

for key, value in cluster_conifg.items():
    print("export CORTEX_{}={}".format(key.upper(), value))
