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
import sys
import traceback
import os


def get_istio_api_gateway_elb_arn(client_elb):
    paginator = client_elb.get_paginator("describe_load_balancers")
    for elb_page in paginator.paginate():
        for elb in elb_page["LoadBalancers"]:
            elb_arn = elb["LoadBalancerArn"]
            elb_tags = client_elb.describe_tags(ResourceArns=[elb_arn])["TagDescriptions"][0][
                "Tags"
            ]
            for tag in elb_tags:
                if (
                    tag["Key"] == "kubernetes.io/service-name"
                    and tag["Value"] == "istio-system/ingressgateway-apis"
                ):
                    return elb_arn
    raise Exception("Could not find ingressgateway-apis ELB")


def get_listener_arn(elb_arn, client_elb):
    paginator = client_elb.get_paginator("describe_listeners")
    for listener_page in paginator.paginate(LoadBalancerArn=elb_arn):
        for listener in listener_page["Listeners"]:
            if listener["Port"] == 443:
                return listener["ListenerArn"]
    raise Exception("Could not find ELB port 443 listener")


def create_gateway_intregration(api_id, vpc_link_id):
    client_elb = boto3.client("elbv2", region_name=os.environ["CORTEX_REGION"])
    client_apigateway = boto3.client("apigatewayv2", region_name=os.environ["CORTEX_REGION"])

    elb_arn = get_istio_api_gateway_elb_arn(client_elb)
    listener_arn = get_listener_arn(elb_arn, client_elb)

    client_apigateway.create_integration(
        ApiId=api_id,
        ConnectionId=vpc_link_id,
        ConnectionType="VPC_LINK",
        IntegrationType="HTTP_PROXY",
        IntegrationUri=listener_arn,
        PayloadFormatVersion="1.0",
        IntegrationMethod="ANY",
    )


if __name__ == "__main__":
    api_id = str(sys.argv[1])
    vpc_link_id = str(sys.argv[2])

    try:
        create_gateway_intregration(api_id, vpc_link_id)
    except:
        print("failed to create API Gateway integration")
        traceback.print_exc()
