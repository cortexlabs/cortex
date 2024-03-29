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


def get_operator_load_balancer_v2(cluster_name, client_elbv2):
    return _get_load_balancer_v2("operator", cluster_name, client_elbv2)


def get_api_load_balancer_v2(cluster_name, client_elbv2):
    return _get_load_balancer_v2("api", cluster_name, client_elbv2)


def get_api_load_balancer(cluster_name, client_elb):
    return _get_load_balancer("api", cluster_name, client_elb)


def get_api_load_balancer_health(load_balancer_name, client_elb):
    instance_health = client_elb.describe_instance_health(
        LoadBalancerName=load_balancer_name,
    )
    for instance_state in instance_health["InstanceStates"]:
        if instance_state["State"] != "InService":
            return "inactive"
    return "active"


def _get_load_balancer_v2(load_balancer_tag, cluster_name, client_elbv2):
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
                if tags["Key"] == "cortex.dev/load-balancer" and tags["Value"] == load_balancer_tag:
                    foundLoadBalancerTag = True
            if foundClusterNameTag and foundLoadBalancerTag:
                return load_balancers[tag_description["ResourceArn"]]

    raise Exception(f"unable to find {load_balancer_tag} load balancer")


def _get_load_balancer(load_balancer_tag, cluster_name, client_elb):
    paginator = client_elb.get_paginator("describe_load_balancers")
    for load_balancer_page in paginator.paginate(PaginationConfig={"PageSize": 20}):
        load_balancers = {
            load_balancer["LoadBalancerName"]: load_balancer
            for load_balancer in load_balancer_page["LoadBalancerDescriptions"]
        }
        tag_descriptions = client_elb.describe_tags(LoadBalancerNames=list(load_balancers.keys()))[
            "TagDescriptions"
        ]
        for tag_description in tag_descriptions:
            foundClusterNameTag = False
            foundLoadBalancerTag = False
            for tags in tag_description["Tags"]:
                if tags["Key"] == "cortex.dev/cluster-name" and tags["Value"] == cluster_name:
                    foundClusterNameTag = True
                if tags["Key"] == "cortex.dev/load-balancer" and tags["Value"] == load_balancer_tag:
                    foundLoadBalancerTag = True
            if foundClusterNameTag and foundLoadBalancerTag:
                return load_balancers[tag_description["LoadBalancerName"]]

    raise Exception(f"unable to find {load_balancer_tag} load balancer")
