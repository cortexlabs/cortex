# Security

## IAM permissions

### Operator

The operator requires read permissions for any data sources, read and write permissions for the Cortex S3 bucket, and read and write permissions for the Cortex CloudWatch log group. The pre-defined `AmazonS3FullAccess` and `CloudWatchLogsFullAccess` policies cover these permissions, but you can create more limited policies manually.

Note: if you don't already have a Cortex S3 bucket and/or Cortex CloudWatch log group, you will need to add create permissions during installation.

### CLI

In order to connect to the operator via the CLI, you must provide valid AWS credentials for any user with access to the account. No special permissions are required. The CLI can be configured using the command `cortex configure`.

### eksctl

See the [eksctl documentation](https://eksctl.io).

## API access

By default, your APIs will be accessible to all traffic. You can restrict access using AWS security groups. Specifically, you will need to edit the security group with the description: "Security group for Kubernetes ELB <ELB name> (cortex/nginx-controller-apis)".
