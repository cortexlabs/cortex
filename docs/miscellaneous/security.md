# Security

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

_The information on this page assumes you are running Cortex on AWS. If you're only deploying locally, this information does not apply (although AWS credentials can still be passed into your APIs, and can be specified with `cortex env configure local`)_

## Private cluster subnets

By default, instances are created in public subnets and are assigned public IP addresses. You can configure all instances in your cluster to use private subnets by setting `subnet_visibility: private` in your [cluster configuration](../cluster-management/config.md) file before creating your cluster. If private subnets are used, instances will not have public IP addresses, and Cortex will create a NAT gateway to allow outgoing network requests.

## Private APIs

See [networking](../deployments/networking.md) for a discussion of API visibility.

## Private operator

By default, the Cortex cluster operator's load balancer is internet-facing, and therefore publicly accessible (the operator is what the `cortex` CLI connects to). The operator validates that the CLI user is an active IAM user in the same AWS account as the Cortex cluster (see [below](#cli)). Therefore it is usually unnecessary to configure the operator's load balancer to be private, but this can be done by by setting `operator_load_balancer_scheme: internal` in your [cluster configuration](../cluster-management/config.md) file. If you do this, you will need to configure [VPC Peering](../guides/vpc-peering.md) to allow your CLI to connect to the Cortex operator (this will be necessary to run any `cortex` commands).

## AWS Credentials/IAM permissions

If you are not using a sensitive AWS account and do not have a lot of experience with IAM configuration, attaching the built-in `AdministratorAccess` policy to your IAM user will make getting started much easier. If you would like to limit IAM permissions, continue reading.

Cortex uses AWS credentials for 3 main purposes:

1. Spinning up a cluster (credentials with `AdministratorAccess` is recommended)
2. Cluster runtime (see [operator policy](#operator))
3. CLI authentication (no special permissions are required)

### Cluster spin-up

Spinning up Cortex on your AWS account requires more permissions than Cortex needs once it's running. You can specify different credentials for each purpose in two ways (in order of precedence):

1. You can specify `--aws-key` and `--aws-secret` flags with the command `cortex cluster up` to indicate the credentials that will be used to create your cluster. Optionally, you can specify `--cluster-aws-key` and `--cluster-aws-secret` to specify credentials which will be used by the cluster.

2. You can export the environment variables `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` which will be used to create your cluster. Optionally, you can export `CLUSTER_AWS_ACCESS_KEY_ID` and `CLUSTER_AWS_SECRET_ACCESS_KEY` to specify credentials which will be used by the cluster.

In either case, the credentials used when spinning up the cluster will not be used by the cluster itself, and can be safely revoked after the cluster is running. You may need credentials with similar access to run other `cortex cluster` commands, such as `cortex cluster configure`, `cortex cluster info`, and `cortex cluster down`.

It is recommended to use an IAM user with the `AdministratorAccess` policy to create your cluster, since the CLI requires many permissions for this step, and the list of permissions is evolving as Cortex adds new features. If it is not possible to use `AdministratorAccess` in your existing AWS account, you could create a separate AWS account for your Cortex cluster, or you could ask someone within your organization to create the Cortex cluster for you (since `AdministratorAccess` is not required to deploy APIs to your cluster; see [CLI](#cli) below).

### Operator

A process called the Cortex operator runs on your cluster and is responsible for deploying and managing your APIs on the cluster. The operator will use `CLUSTER_AWS_ACCESS_KEY_ID` and `CLUSTER_AWS_SECRET_ACCESS_KEY` as its credentials if specified, otherwise it will default to using `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.

The operator requires read permissions for any S3 bucket containing exported models, read/write permissions for the Cortex S3 bucket, read permissions for ECR, read permissions for ELB, read/write permissions for API Gateway, read/write permissions for CloudWatch metrics, and read/write permissions for the Cortex CloudWatch log group. The policy below may be used to restrict the Operator's access:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": "*"
        },
        {
            "Action": [
                "sts:GetCallerIdentity",
                "ecr:GetAuthorizationToken",
                "ecr:BatchGetImage",
                "elasticloadbalancing:Describe*",
                "apigateway:*",
                "cloudwatch:*",
                "logs:*",
                "sqs:*"
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
