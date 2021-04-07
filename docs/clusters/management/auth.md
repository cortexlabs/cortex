# Auth

## Client

Cortex client uses the default credential provider chain to get credentials for both cluster and api management. Credentials will be read in the following order of precedence:

- environment variables
- the name of the profile specified by `AWS_PROFILE` environment variable
- `default` profile from `~/.aws/credentials`

### Cluster management

It is recommended that your AWS credentials have AdminstratorAccess when running `cortex cluster *` commands. For the advanced users, see [minimum IAM policy](#minimum-iam-policy) section below to see view the minimum permissions required to run `cortex cluster *` commands.

After spinning up a cluster using `cortex cluster up`, the IAM entity user or role that created the cluster is automatically granted `system:masters` permission to the cluster's RBAC. Make sure to keep track of which IAM entity originally created the cluster.

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

Cortex client relies on AWS IAM to authenticate requests (e.g. `cortex deploy`, `cortex get`) to a cluster on AWS. The client will include a get-caller-identity request that has been signed with the credentials from the default credential provider chain along with original request. The operator executes the presigned request to verify that credentials are valid and belong to the same account as the IAM entity of the cluster.

AWS credentials required to authenticate cortex client requests to the operator don't require any permissions. However, managing the cluster using `cortex cluster *` commands do require permissions.

## Authorizing your APIs

When spinning up a cortex cluster, you can provide additional policies to authorize your APIs to access AWS resources by creating a policy and adding it to the `iam_policy_arns` list.

If you already have a cluster running and would like to add additional permissions, you can update the policy that is created automatically during `cortex cluster up`. In [IAM console](https://console.aws.amazon.com/iam/home?policies#/policies) search for `cortex-<cluster_name>-<region>` to find the policy that has been attached to your cluster. Adding more permissions to this policy will automatically give more access to all of your Cortex APIs.

_NOTE: The policy created during `cortex cluster up` will automatically be deleted during `cortex cluster down`. It is recommended to create your own policies that can be specified in `iam_policy_arns` field in cluster configuration. The precreated policy should only be updated for development and testing purposes._

## Minimum IAM Policy

The policy listed below is the minimum permissions required to manage a Cortex cluster and its dependencies.

Replace the following $CORTEX_CLUSTER_NAME, $CORTEX_ACCOUNT_ID, $CORTEX_REGION with their respective values in the policy template below.

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
                "logs:ListTagsLogGroup",
                "iam:GetRole",
                "logs:TagLogGroup",
                "ssm:GetParameters",
                "ssm:GetParameter",
                "logs:CreateLogGroup"
            ],
            "Resource": [
                "arn:aws:ssm:*:$CORTEX_ACCOUNT_ID:parameter/aws/*",
                "arn:aws:ssm:*::parameter/aws/*",
                "arn:aws:logs:$CORTEX_REGION:$CORTEX_ACCOUNT_ID:log-group:$CORTEX_CLUSTER_NAME",
                "arn:aws:iam::$CORTEX_ACCOUNT_ID:role/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "iam:CreateInstanceProfile",
                "logs:ListTagsLogGroup",
                "logs:DescribeLogStreams",
                "iam:TagRole",
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
                "iam:DeleteInstanceProfile",
                "iam:GetRole",
                "iam:GetInstanceProfile",
                "iam:DeleteRole",
                "iam:ListInstanceProfiles",
                "logs:CreateLogGroup",
                "logs:PutLogEvents",
                "iam:CreateOpenIDConnectProvider",
                "iam:GetOpenIDConnectProvider",
                "iam:GetRolePolicy"
            ],
            "Resource": [
                "arn:aws:iam::$CORTEX_ACCOUNT_ID:instance-profile/eksctl-*",
                "arn:aws:iam::$CORTEX_ACCOUNT_ID:role/eksctl-*",
                "arn:aws:iam::$CORTEX_ACCOUNT_ID:role/aws-service-role/eks-nodegroup.amazonaws.com/AWSServiceRoleForAmazonEKSNodegroup",
                "arn:aws:iam::$CORTEX_ACCOUNT_ID:role/eksctl-managed-*",
                "arn:aws:iam::$CORTEX_ACCOUNT_ID:oidc-provider/*",
                "arn:aws:logs:$CORTEX_REGION:$CORTEX_ACCOUNT_ID:log-group:$CORTEX_CLUSTER_NAME:*"
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
            "Resource": "arn:aws:iam::$CORTEX_ACCOUNT_ID:policy/cortex-*"
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
                "kms:CreateGrant"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "sqs:*",
            "Resource": "arn:aws:sqs:$CORTEX_REGION:$CORTEX_ACCOUNT_ID:cx-*"
        },
        {
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": "arn:aws:s3:::$CORTEX_CLUSTER_NAME*"
        },
        {
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": "arn:aws:s3:::$CORTEX_CLUSTER_NAME*/*"
        }
    ]
}
```
