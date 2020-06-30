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

import boto3
import os


def get_api_gateway_id():
    cluster_name = os.environ["CORTEX_CLUSTER_NAME"]
    region = os.environ["CORTEX_REGION"]
    client_apigateway = boto3.client("apigatewayv2", region_name=region)

    paginator = client_apigateway.get_paginator("get_apis")
    for api_gateway_page in paginator.paginate():
        for api_gateway in api_gateway_page["Items"]:
            if api_gateway["Tags"].get("cortex.dev/cluster-name") == cluster_name:
                return api_gateway["ApiId"]

    raise Exception(
        f"your cluster's api gateway (in {region} with tag cortex.dev/cluster-name={cluster_name}) does not exist"
    )


if __name__ == "__main__":
    print(get_api_gateway_id(), end="")
