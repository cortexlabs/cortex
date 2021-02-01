# Security

## Private cluster subnets

By default, instances are created in public subnets and are assigned public IP addresses. You can configure all instances in your cluster to use private subnets by setting `subnet_visibility: private` in your [cluster configuration](install.md) file before creating your cluster. If private subnets are used, instances will not have public IP addresses, and Cortex will create a NAT gateway to allow outgoing network requests.

## Private APIs

See [networking](networking/index.md) for a discussion of API visibility.

## Private operator

By default, the Cortex cluster operator's load balancer is internet-facing, and therefore publicly accessible (the operator is what the `cortex` CLI connects to). The operator validates that the CLI user is an active IAM user in the same AWS account as the Cortex cluster (see [below](#cli)). Therefore it is usually unnecessary to configure the operator's load balancer to be private, but this can be done by by setting `operator_load_balancer_scheme: internal` in your [cluster configuration](install.md) file. If you do this, you will need to configure [VPC Peering](networking/vpc-peering.md) to allow your CLI to connect to the Cortex operator (this will be necessary to run any `cortex` commands).

## IAM permissions

If you are not using a sensitive AWS account and do not have a lot of experience with IAM configuration, attaching the built-in `AdministratorAccess` policy to your IAM user will make getting started much easier. If you would like to limit IAM permissions, continue reading.

Cortex uses AWS credentials for 3 main purposes:

1. Spinning up a cluster (credentials with `AdministratorAccess` is recommended)
2. Cluster runtime (see [operator policy](#operator))
3. CLI authentication (no special permissions are required)

### Cluster spin-up

You can specify credentials for spinning up the cluster in four ways (in order of precedence):

1. You can specify `--aws-key` and `--aws-secret` flags with the command `cortex cluster up` to indicate the credentials that will be used to create your cluster. Optionally, you can specify `--cluster-aws-key` and `--cluster-aws-secret` to specify credentials which will be used by the cluster. When all four flags are specified, the credentials used when spinning up the cluster will not be used by the cluster itself. If `--cluster-aws-key` and `--cluster-aws-secret` flags are not specified, then they'll get set to the values of `--aws-key` and `--aws-secret` respectively.

2. From your Cortex CLI cluster cache. If a cluster with the same name and region has existed before, the AWS credentials of that will now be used for the current creation of the cluster.

3. You can export the environment variables `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` which will be used to create your cluster. Optionally, you can export `CLUSTER_AWS_ACCESS_KEY_ID` and `CLUSTER_AWS_SECRET_ACCESS_KEY` to specify credentials which will be used by the cluster. When all four environment variables are set, the credentials used when spinning up the cluster will not be used by the cluster itself. If `CLUSTER_AWS_ACCESS_KEY_ID` and `CLUSTER_AWS_SECRET_ACCESS_KEY` environment variables are not set, then they'll get set to the values of `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` respectively.

4. You can configure the shared AWS credentials with `aws configure` which will then be used to create your cluster.

5. You can specify the AWS credentials `"aws access key id"` and `"aws secret access key"` at the CLI's prompt when requested.

It is recommended to use an IAM user with the `AdministratorAccess` policy to create your cluster, since the CLI requires many permissions for this step, and the list of permissions is evolving as Cortex adds new features. If it is not possible to use `AdministratorAccess` in your existing AWS account, you could create a separate AWS account for your Cortex cluster, or you could ask someone within your organization to create the Cortex cluster for you (since `AdministratorAccess` is not required to deploy APIs to your cluster; see [CLI](#cli) below).

### Operator

A process called the Cortex operator runs on your cluster and is responsible for deploying and managing your APIs on the cluster. The operator will use the designated cluster credentials (e.g. `--cluster-aws-key` or `$CLUSTER_AWS_ACCESS_KEY_ID`) if specified, otherwise it will default to using the credentials used to spin up the cluster (e.g. `--aws-key` or `$AWS_ACCESS_KEY_ID`).

The operator requires read permissions for any S3 bucket containing exported models, read/write permissions for the Cortex S3 bucket, read permissions for ECR, read permissions for ELB, read/write permissions for CloudWatch metrics, and read/write permissions for the Cortex CloudWatch log group. The policy below may be used to restrict the Operator's access:

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

### Running `cortex cluster` commands from different IAM users

By default, the `cortex cluster *` commands can only be executed by the IAM user who created the Cortex cluster. To grant access to additional IAM users, follow these steps:

1. Install `eksctl` by following these [instructions](https://eksctl.io/introduction/#installation).

1. Determine the ARN of the IAM user that you would like to grant access to. You can get the ARN via the [IAM dashboard](https://console.aws.amazon.com/iam/home#/users), or by running `aws iam get-user` on a machine that is authenticated as the IAM user (or `AWS_ACCESS_KEY_ID=*** AWS_SECRET_ACCESS_KEY=*** aws iam get-user` on any machine, using the credentials of the IAM user). The ARN should look similar to `arn:aws:iam::764403040417:user/my-username`.

1. Set the following environment variables:

    ```bash
    CREATOR_AWS_ACCESS_KEY_ID=***      # access key ID for the IAM user that created the cluster
    CREATOR_AWS_SECRET_ACCESS_KEY=***  # secret access key for the IAM user that created the cluster
    NEW_USER_ARN=***                   # ARN of the IAM user to grant access to
    CLUSTER_NAME=***                   # the name of your cortex cluster (will be "cortex" unless you specified a different name in your cluster configuration file)
    CLUSTER_REGION=***                 # the region of your cortex cluster
    ```

1. Run the following command:

    ```bash
    AWS_ACCESS_KEY_ID=$CREATOR_AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY=$CREATOR_AWS_SECRET_ACCESS_KEY eksctl create iamidentitymapping --region $CLUSTER_REGION --cluster $CLUSTER_NAME --arn $NEW_USER_ARN --group system:masters --username $NEW_USER_ARN
    ```

1. To revoke access in the future, run:

    ```bash
    AWS_ACCESS_KEY_ID=$CREATOR_AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY=$CREATOR_AWS_SECRET_ACCESS_KEY eksctl delete iamidentitymapping --region $CLUSTER_REGION --cluster $CLUSTER_NAME --arn $NEW_USER_ARN --all
    ```
