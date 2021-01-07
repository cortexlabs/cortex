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

import boto3
import os

from helpers import get_api_load_balancer


def get_api_load_balancer_state():
    cluster_name = os.environ["CORTEX_CLUSTER_NAME"]
    region = os.environ["CORTEX_REGION"]

    client_elbv2 = boto3.client("elbv2", region_name=region)

    load_balancer = get_api_load_balancer(cluster_name, client_elbv2)
    return load_balancer["State"]["Code"]


if __name__ == "__main__":
    print(get_api_load_balancer_state(), end="")
