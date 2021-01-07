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
import json

from helpers import get_operator_load_balancer


def get_operator_target_group_status():
    cluster_name = os.environ["CORTEX_CLUSTER_NAME"]
    region = os.environ["CORTEX_REGION"]

    client_elbv2 = boto3.client("elbv2", region_name=region)

    load_balancer_arn = get_operator_load_balancer(cluster_name, client_elbv2)["LoadBalancerArn"]
    target_group_arn = get_load_balancer_https_target_group_arn(load_balancer_arn, client_elbv2)
    return get_target_health(target_group_arn, client_elbv2)


def get_load_balancer_https_target_group_arn(load_balancer_arn, client_elbv2):
    paginator = client_elbv2.get_paginator("describe_listeners")
    for listener_page in paginator.paginate(LoadBalancerArn=load_balancer_arn):
        for listener in listener_page["Listeners"]:
            if listener["Port"] == 443:
                return listener["DefaultActions"][0]["TargetGroupArn"]

    raise Exception(
        f"unable to find https target group for operator load balancer ({load_balancer_arn})"
    )


def get_target_health(target_group_arn, client_elbv2):
    response = client_elbv2.describe_target_health(TargetGroupArn=target_group_arn)
    for health_description in response["TargetHealthDescriptions"]:
        if health_description["TargetHealth"]["State"] == "healthy":
            return "healthy"

    return json.dumps(response["TargetHealthDescriptions"])


if __name__ == "__main__":
    print(get_operator_target_group_status(), end="")
