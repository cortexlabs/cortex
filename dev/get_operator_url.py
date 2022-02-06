# Copyright 2022 Cortex Labs, Inc.
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
import boto3


def main():
    cluster_name = sys.argv[1]
    region = sys.argv[2]
    operator_url = get_operator_url(cluster_name, region)
    print("https://" + operator_url)


def get_operator_url(cluster_name, region):
    client_elbv2 = boto3.client("elbv2", region_name=region)

    paginator = client_elbv2.get_paginator("describe_load_balancers")
    for load_balancer_page in paginator.paginate(PaginationConfig={"PageSize": 20}):
        load_balancers = {
            load_balancer["LoadBalancerArn"]: load_balancer
            for load_balancer in load_balancer_page["LoadBalancers"]
        }
        tag_descriptions = client_elbv2.describe_tags(ResourceArns=list(load_balancers.keys()))[
            "TagDescriptions"
        ]
        for tag_description in tag_descriptions:
            foundClusterNameTag = False
            foundLoadBalancerTag = False
            for tags in tag_description["Tags"]:
                if tags["Key"] == "cortex.dev/cluster-name" and tags["Value"] == cluster_name:
                    foundClusterNameTag = True
                if tags["Key"] == "cortex.dev/load-balancer" and tags["Value"] == "operator":
                    foundLoadBalancerTag = True
            if foundClusterNameTag and foundLoadBalancerTag:
                load_balancer = load_balancers[tag_description["ResourceArn"]]
                return load_balancer["DNSName"]


# usage: python get_operator_url.py CLUSTER_NAME REGION
if __name__ == "__main__":
    main()
