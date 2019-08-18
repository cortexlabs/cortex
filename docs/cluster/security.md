# Security

## IAM permissions

If you are not using a sensitive AWS account and do not have a lot of experience with IAM configuration, attaching the existing policy `AdministratorAccess` to your IAM user will make getting started much easier.

### Operator

The operator requires read permissions for any S3 bucket containing exported models, read and write permissions for the Cortex S3 bucket, read and write permissions for the Cortex CloudWatch log group, and read and write permissions for CloudWatch metrics. The policy below may be used to restrict the Operator's access:

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
                "logs:*"
            ],
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
```

### CLI

In order to connect to the operator via the CLI, you must provide valid AWS credentials for any user with access to the account. No special permissions are required. The CLI can be configured using the command `cortex configure`.

## API access

By default, your Cortex APIs will be accessible to all traffic. You can restrict access using AWS security groups. Specifically, you will need to edit the security group with the description: "Security group for Kubernetes ELB <ELB name> (istio-system/apis-ingressgateway)".
