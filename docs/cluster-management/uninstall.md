# Uninstall

## Prerequisites

1. [AWS credentials](aws.md)
2. [Docker](https://docs.docker.com/install)
3. [Cortex CLI](../install.md)
4. [AWS CLI](https://aws.amazon.com/cli)

## Uninstalling Cortex

```bash
# spin down the cluster
cortex cluster down

# uninstall the CLI
sudo rm /usr/local/bin/cortex
rm -rf ~/.cortex
```

If you modified your bash profile, you may wish to remove `source <(cortex completion)`.

## Cleaning up AWS

```bash
# set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# delete the S3 bucket
aws s3 ls
aws s3 rb --force s3://<bucket-name>

# delete the log group
aws logs describe-log-groups --log-group-name-prefix=<log_group_name> --query logGroups[*].[logGroupName] --output text | xargs -I {} aws logs delete-log-group --log-group-name {}
```

