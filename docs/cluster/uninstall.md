# Uninstall

## Prerequisites

1. [AWS credentials](aws.md)
2. [Docker](https://docs.docker.com/install)

## Uninstalling Cortex

<!-- CORTEX_VERSION_MINOR -->

```bash
# Download
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex.sh

# Change permissions
chmod +x cortex.sh

# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Uninstall Cortex
./cortex.sh uninstall
```

## Uninstalling the CLI

```bash
# Uninstall the Cortex CLI
./cortex.sh uninstall cli
```

## Cleaning up AWS

```bash
# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Delete the S3 bucket
aws s3 ls
aws s3 rb --force s3://<bucket-name>

# Delete the log group
aws logs describe-log-groups --log-group-name-prefix=<log_group_name> --query logGroups[*].[logGroupName] --output text | xargs -I {} aws logs delete-log-group --log-group-name {}
```
