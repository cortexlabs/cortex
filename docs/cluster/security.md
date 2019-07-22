# Security

## IAM permissions

If you are not using a sensitive AWS account and do not have a lot of experience with IAM configuration, attaching the existing policy `AdministratorAccess` to your IAM user will make getting started much easier.

### Operator

The operator requires read permissions for any data sources, read and write permissions for the Cortex S3 bucket, and read and write permissions for the Cortex CloudWatch log group. The pre-defined `AmazonS3FullAccess` and `CloudWatchLogsFullAccess` policies cover these permissions, but you can create more limited policies manually.

If you don't already have a Cortex S3 bucket and/or Cortex CloudWatch log group, you will need to add create permissions during installation.

### CLI

In order to connect to the operator via the CLI, you must provide valid AWS credentials for any user with access to the account. No special permissions are required. The CLI can be configured using the command `cortex configure`.

## API access

By default, your Cortex APIs will be accessible to all traffic. You can restrict access using AWS security groups. Specifically, you will need to edit the security group with the description: "Security group for Kubernetes ELB <ELB name> (cortex/nginx-controller-apis)".
