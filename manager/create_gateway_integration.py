import boto3
import sys
import traceback


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


def create_gateway_intregration(api_id, vpc_link_id):
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


if __name__ == "__main__":
    api_id = str(sys.argv[1])
    vpc_link_id = str(sys.argv[2])
    region = str(sys.argv[3])
    client_elb = boto3.client("elbv2", region_name=region)
    client_apigateway = boto3.client("apigatewayv2", region_name=region)
    try:
        create_gateway_intregration(api_id, vpc_link_id)
    except:
        print("failed to create API Gateway integration")
        traceback.print_exc()
