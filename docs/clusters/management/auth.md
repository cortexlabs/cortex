# Auth

## Client

The Cortex CLI and Python client use the default credential provider chain to get credentials for cluster and api management. Credentials will be read in the following order of precedence:

- environment variables
- the name of the profile specified by `AWS_PROFILE` environment variable
- `default` profile from `~/.aws/credentials`

### Cluster management

It is recommended that your AWS credentials have AdministratorAccess when running `cortex cluster *` commands. If you are unable to use AdministratorAccess, see the [minimum IAM policy](#minimum-iam-policy) below for the minimum permissions required to run `cortex cluster *` commands.

After spinning up a cluster using `cortex cluster up`, the IAM user or role that created the cluster is automatically granted `system:masters` permission to the cluster's RBAC. Make sure to keep track of which IAM entity originally created the cluster.

#### Running `cortex cluster` commands from different IAM users

By default, the `cortex cluster *` commands can only be executed by the IAM user who created the cluster. To grant access to additional IAM users, follow these steps:

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

### API management

The Cortex CLI and Python client rely on AWS IAM to authenticate requests to a cluster on AWS (e.g. `cortex deploy`, `cortex get`). AWS credentials required to authenticate Cortex client requests to the operator don't require any specific permissions; they must only be valid credentials within the same AWS account as the Cortex cluster. However, managing the cluster (i.e. running `cortex cluster *` commands) does require permissions.

## Authorizing your APIs

When spinning up a cortex cluster, you can provide additional policies to authorize your APIs to access AWS resources by creating a policy and adding it to the `iam_policy_arns` list in your cluster configuration file.

If you already have a cluster running and would like to add additional permissions, you can update the policy that is created automatically during `cortex cluster up`. In the [IAM console](https://console.aws.amazon.com/iam/home?policies#/policies), search for `cortex-<cluster_name>-<region>` to find the policy that has been attached to your cluster. Adding more permissions to this policy will automatically give more access to all of your Cortex APIs.

_NOTE: The policy created during `cortex cluster up` will automatically be deleted during `cortex cluster down`. It is recommended to create your own policies that can be specified in `iam_policy_arns` field in cluster configuration. The precreated policy should only be updated for development and testing purposes._

## Minimum IAM Policy

The policy shown below contains the minimum permissions required to manage a Cortex cluster (i.e. via `cortex cluster *` commands).

Replace the following placeholders with their respective values in the policy template below: `$CORTEX_CLUSTER_NAME`, `$CORTEX_ACCOUNT_ID`, `$CORTEX_REGION`.

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "iam:CreateServiceLinkedRole",
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "iam:AWSServiceName": [
                        "autoscaling.amazonaws.com",
                        "ec2scheduled.amazonaws.com",
                        "elasticloadbalancing.amazonaws.com",
                        "spot.amazonaws.com",
                        "spotfleet.amazonaws.com",
                        "transitgateway.amazonaws.com"
                    ]
                }
            }
        },
        {
            "Effect": "Allow",
            "Action": "iam:CreateServiceLinkedRole",
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "iam:AWSServiceName": [
                        "eks.amazonaws.com",
                        "eks-nodegroup.amazonaws.com",
                        "eks-fargate.amazonaws.com"
                    ]
                }
            }
        },
        {
            "Effect": "Allow",
            "Action": [
                "logs:ListTagsForResource",
                "iam:GetRole",
                "logs:TagLogGroup",
                "ssm:GetParameters",
                "ssm:GetParameter",
                "logs:CreateLogGroup"
            ],
            "Resource": [
                "arn:*:ssm:*:$CORTEX_ACCOUNT_ID:parameter/aws/*",
                "arn:*:ssm:*::parameter/aws/*",
                "arn:*:logs:$CORTEX_REGION:$CORTEX_ACCOUNT_ID:log-group:$CORTEX_CLUSTER_NAME",
                "arn:*:iam::$CORTEX_ACCOUNT_ID:role/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "iam:CreateInstanceProfile",
                "logs:ListTagsForResource",
                "logs:DescribeLogStreams",
                "iam:TagRole",
                "iam:GetPolicy",
                "iam:CreatePolicy",
                "iam:DeletePolicy",
                "iam:ListPolicyVersions",
                "iam:RemoveRoleFromInstanceProfile",
                "iam:CreateRole",
                "iam:AttachRolePolicy",
                "iam:PutRolePolicy",
                "iam:AddRoleToInstanceProfile",
                "iam:ListInstanceProfilesForRole",
                "iam:PassRole",
                "logs:CreateLogStream",
                "iam:DetachRolePolicy",
                "logs:TagLogGroup",
                "iam:ListAttachedRolePolicies",
                "iam:DeleteRolePolicy",
                "iam:DeleteOpenIDConnectProvider",
                "iam:TagOpenIDConnectProvider",
                "iam:DeleteInstanceProfile",
                "iam:GetRole",
                "iam:GetInstanceProfile",
                "iam:DeleteRole",
                "iam:ListInstanceProfiles",
                "logs:CreateLogGroup",
                "logs:PutLogEvents",
                "logs:DeleteLogGroup",
                "iam:CreateOpenIDConnectProvider",
                "iam:GetOpenIDConnectProvider",
                "iam:GetRolePolicy"
            ],
            "Resource": [
                "arn:*:iam::$CORTEX_ACCOUNT_ID:instance-profile/eksctl-*",
                "arn:*:iam::$CORTEX_ACCOUNT_ID:role/eksctl-*",
                "arn:*:iam::$CORTEX_ACCOUNT_ID:policy/eksctl-*",
                "arn:*:iam::$CORTEX_ACCOUNT_ID:role/aws-service-role/eks-nodegroup.amazonaws.com/AWSServiceRoleForAmazonEKSNodegroup",
                "arn:*:iam::$CORTEX_ACCOUNT_ID:role/eksctl-managed-*",
                "arn:*:iam::$CORTEX_ACCOUNT_ID:oidc-provider/*",
                "arn:*:logs:$CORTEX_REGION:$CORTEX_ACCOUNT_ID:log-group:$CORTEX_CLUSTER_NAME:*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "iam:CreatePolicy",
                "iam:GetPolicyVersion",
                "iam:ListPolicyVersions",
                "iam:DeletePolicy",
                "iam:CreatePolicyVersion",
                "iam:DeletePolicyVersion"
            ],
            "Resource": "arn:*:iam::$CORTEX_ACCOUNT_ID:policy/cortex-*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "sqs:ListQueues",
                "iam:GetPolicy",
                "ecr:GetAuthorizationToken",
                "cloudformation:*",
                "elasticloadbalancing:*",
                "autoscaling:*",
                "cloudwatch:*",
                "ecr:BatchGetImage",
                "kms:DescribeKey",
                "ec2:*",
                "sts:GetCallerIdentity",
                "eks:*",
                "kms:CreateGrant",
                "acm:DescribeCertificate",
                "servicequotas:ListServiceQuotas",
                "logs:PutRetentionPolicy"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "sqs:*",
            "Resource": "arn:*:sqs:$CORTEX_REGION:$CORTEX_ACCOUNT_ID:cx-*"
        },
        {
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": "arn:*:s3:::$CORTEX_CLUSTER_NAME*"
        },
        {
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": "arn:*:s3:::$CORTEX_CLUSTER_NAME*/*"
        }
    ]
}
```
