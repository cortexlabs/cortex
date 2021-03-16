# Auth

## Cortex Client

Cortex client uses the default credential provider chain to get credentials. Credentials will be read in the following order of precedence:

- environment variables
- the name of the profile specified by `AWS_PROFILE` environment variable
- `default` profile from `~/.aws/credentials`

### API Management

Cortex client relies on AWS IAM to authenticate requests (e.g. `cortex deploy`, `cortex get`) to a cluster on AWS. The client will include a get-caller-identity request that has been signed with the credentials from the default credential provider chain along with original request. The operator executes the presigned request to verify that credentials are valid and belong to the same account as the IAM entity of the cluster.

AWS credentials required to authenticate cortex client requests to the operator don't require any permissions. However, managing the cluster using `cortex cluster *` commands do require permissions.

### Cluster Management

It is recommended that your AWS credentials have AdminstratorAccess before running `cortex cluster *` commands.

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

## Authorizing your APIs

When spinning up a cortex cluster, you can provide additional policies to authorize your APIs to access AWS resources by creating a policy and adding it to the `iam_policy_arns` list.

If you already have a cluster running and would like to add additional permissions, you can update the policy that is created automatically during `cortex cluster up`. In [IAM console](https://console.aws.amazon.com/iam/home?policies#/policies) search for `cortex-<cluster_name>-<region>` to find the policy that has been attached to your cluster. Adding more permissions to this policy will automatically give more access to all of your Cortex APIs.

_NOTE: The policy created during `cortex cluster up` will automatically be deleted during `cortex cluster down`. It is recommended to create your own policies that can be specified in `iam_policy_arns` field in cluster configuration. The precreated policy should only be updated for development and testing purposes._

`cortex cluster up` will create a default policy which is the minimum set of IAM permissions to run your APIs and various processes on the cluster such as the operator and fluent-bit. Fields from your AWS account and cluster configuration will be used to populate the policy below.

```json
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": [
				"sts:GetCallerIdentity",
				"ecr:GetAuthorizationToken",
				"ecr:BatchGetImage",
				"sqs:ListQueues"
			],
			"Effect": "Allow",
			"Resource": "*"
		},
		{
			"Effect": "Allow",
			"Action": "sqs:*",
			"Resource": "arn:aws:sqs:{{ .Region }}:{{ .AccountID }}:cx-*"
		},
		{
			"Effect": "Allow",
			"Action": "s3:*",
			"Resource": "arn:aws:s3:::{{ .Bucket }}"
		},
		{
			"Effect": "Allow",
			"Action": "s3:*",
			"Resource": "arn:aws:s3:::{{ .Bucket }}/*"
		},
		{
			"Effect": "Allow",
			"Action": [
				"logs:CreateLogStream",
				"logs:DescribeLogStreams",
				"logs:PutLogEvents",
				"logs:CreateLogGroup"
			],
			"Resource": "arn:aws:logs:{{ .Region }}:{{ .AccountID }}:log-group:{{ .LogGroup }}:*"
		},
		{
			"Effect": "Allow",
			"Action": "logs:CreateLogGroup",
			"Resource": "arn:aws:logs:{{ .Region }}:{{ .AccountID }}:log-group:{{ .LogGroup }}"
		}
	]
}
```
