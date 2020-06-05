# Security

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

_The information on this page assumes you are running Cortex on AWS. If you're only deploying locally, this information does not apply (although AWS credentials can still be passed into your APIs, and can be specified with `cortex env configure local`)_

## Private cluster

By default, instances are created in public subnets and are assigned public IP addresses. You can configure all instances in your cluster to use private subnets by setting `subnet_visibility: private` in your [cluster configuration](../cluster-management/config.md) file before creating your cluster. If private subnets are used, instances will not have public IP addresses, and Cortex will create a NAT gateway to allow outgoing network requests.

By default, the API load balancer is internet-facing. You can configure your API load balancer to be internal by setting `api_load_balancer_scheme: internal` in your [cluster configuration](../cluster-management/config.md) file (before creating your cluster).

Regardless of the API load balancer's configuration, APIs are deployed with an internet-facing API Gateway by default (the API Gateway forwards requests to the API load balancer). Each API can be independently configured to not create the API Gateway endpoint by setting `api_gateway: none` in the `networking` field of the [api configuration](../deployments/api-configuration). If the API Gateway endpoint is not created, your API can still be accessed via the API load balancer; `cortex get API_NAME` will show your load balancer endpoint if API Gateway is disabled.

If you'd like to force traffic to go through your public API Gateway endpoint, set `api_load_balancer_scheme: internal` in your [cluster configuration](../cluster-management/config.md) file before creating your cluster.

If you'd like to ensure that an API is not publicly accessible, set `api_load_balancer_scheme: internal` in your [cluster configuration](../cluster-management/config.md) file, and set `api_gateway: none` in the `networking` field of your [api configuration](../deployments/api-configuration). If you do this, you will need to configure [VPC Peering](../guides/vpc-peering.md) to make prediction requests to your APIs.

By default, the Cortex cluster operator's load balancer is internet-facing, and therefore publicly accessible (the operator is what the `cortex` CLI connects to). The operator validates that the CLI user is an active IAM user in the same AWS account as the Cortex cluster (see [below](#cli)). Therefore it is usually unnecessary to configure the operator's load balancer to be internal, but this can be done by by setting `operator_load_balancer_scheme: internal` in your [cluster configuration](../cluster-management/config.md) file. If you do this, you will need to configure [VPC Peering](../guides/vpc-peering.md) to allow your CLI to connect to the Cortex operator (this will be necessary to run any `cortex` commands).

## IAM permissions

If you are not using a sensitive AWS account and do not have a lot of experience with IAM configuration, attaching the built-in `AdministratorAccess` policy to your IAM user will make getting started much easier. If you would like to limit IAM permissions, continue reading.

### Cluster spin-up

Spinning up Cortex on your AWS account requires more permissions that Cortex needs once it's running. You can specify different credentials for each purpose in two ways:

1. You can export the environment variables `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` which will be used to create your cluster, and export `CORTEX_AWS_ACCESS_KEY_ID` and `CORTEX_AWS_SECRET_ACCESS_KEY` which will be used by the cluster.

2. If you are using a cluster configuration file (e.g. `cluster.yaml`), you can set the fields `aws_access_key_id` and `aws_secret_access_key` which will be used to create your cluster, and set `cortex_aws_access_key_id` and `cortex_aws_secret_access_key` which will be used by the cluster.

In either case, the credentials used when spinning up the cluster will not be used by the cluster itself, and can be safely revoked after the cluster is running. You may need credentials with similar access to run other `cortex cluster` commands, such as `cortex cluster configure`, `cortex cluster info`, and `cortex cluster down`.

It is recommended to use an IAM user with the `AdministratorAccess` policy to create your cluster, since the CLI requires many permissions for this step, and the list of permissions is evolving as Cortex adds new features.

### Operator

The operator requires read permissions for any S3 bucket containing exported models, read and write permissions for the Cortex S3 bucket, read and write permissions for the Cortex CloudWatch log group, read and write permissions for CloudWatch metrics, and read permissions for ECR. The policy below may be used to restrict the Operator's access:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "sts:GetCallerIdentity"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": "*"
        },
        {
            "Action": [
                "cloudwatch:*",
                "logs:*",
                "ecr:GetAuthorizationToken"
            ],
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
```

It is possible to further restrict access by limiting access to particular resources (e.g. allowing access to only the bucket containing your models and the cortex bucket).

### CLI

In order to connect to the operator via the CLI, you must provide valid AWS credentials for any user with access to the account. No special permissions are required. The CLI can be configured using the `cortex env configure ENVIRONMENT_NAME` command (e.g. `cortex env configure aws`).

## HTTPS

All APIs are deployed with a public HTTPS API Gateway endpoint by default. If you disable API Gateway for an API (by setting `api_gateway: none` in the `networking` field of the [api configuration](../deployments/api-configuration)), you will be making prediction requests directly to the API load balancer. The SSL certificate on the load balancer is autogenerated during installation using `localhost` as the Common Name (CN). Therefore, clients will need to skip certificate verification (e.g. `curl -k`) when making HTTPS requests to the load balancer, or simply make HTTP requests.

In addition, you can create an SSL certificate for your custom domain, and configure Cortex to use this certificate (see our instructions for [setting up a custom domain](../guides/custom-domain.md)).
