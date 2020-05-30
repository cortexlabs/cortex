import boto3
import sys

client_elb = boto3.client("elbv2")
client_apigateway = boto3.client("apigatewayv2")
api_id = sys.argv[1]
vpc_link_id = sys.argv[2]


def get_istio_api_gateway_elb_arn():
    for elb in client_elb.describe_load_balancers()["LoadBalancers"]:
        elb_arn = elb["LoadBalancerArn"]
        elb_tags = client_elb.describe_tags(ResourceArns=[elb_arn])["TagDescriptions"][0]["Tags"]
        for tag in elb_tags:
            if (
                tag["Key"] == "kubernetes.io/service-name"
                and tag["Value"] == "istio-system/ingressgateway-apis"
            ):
                return elb_arn
    return ""


def get_listener_arn(elb_arn):
    listeners = client_elb.describe_listeners(LoadBalancerArn=elb_arn)["Listeners"]
    for listener in listeners:
        if listener["Port"] == 80:
            return listener["ListenerArn"]
    return ""


elb_arn = get_istio_api_gateway_elb_arn()
listener80_arn = get_listener_arn(elb_arn)
client_apigateway.create_integration(
    ApiId=api_id,
    ConnectionId=vpc_link_id,
    ConnectionType="VPC_LINK",
    IntegrationType="HTTP_PROXY",
    IntegrationUri=listener80_arn,
    PayloadFormatVersion="1.0",
    IntegrationMethod="ANY",
)
