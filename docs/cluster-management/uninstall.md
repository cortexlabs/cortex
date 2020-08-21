# Uninstall

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Prerequisites

1. [AWS credentials](aws-credentials.md)
2. [Docker](https://docs.docker.com/install)
3. [Cortex CLI](install.md)
4. [AWS CLI](https://aws.amazon.com/cli)

## Uninstalling Cortex

```bash
# spin down the cluster
cortex cluster down

# uninstall the CLI
sudo rm /usr/local/bin/cortex
rm -rf ~/.cortex
```

If you modified your bash profile, you may wish to remove `source <(cortex completion bash)` from it (or remove `source <(cortex completion zsh)` for `zsh`).

## Cleaning up AWS

Since you may wish to have access to your data after spinning down your cluster, Cortex's bucket and log groups are not automatically deleted when running `cortex cluster down`.

To delete them:

```bash
# set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# identify the name of your cortex S3 bucket
aws s3 ls

# delete the S3 bucket
aws s3 rb --force s3://<bucket>

# delete the log group (replace <log_group> with what was configured during installation, default: cortex)
aws logs describe-log-groups --log-group-name-prefix=<log_group> --query logGroups[*].[logGroupName] --output text | xargs -I {} aws logs delete-log-group --log-group-name {}
```

If you've configured a custom domain for your APIs, you may wish to remove the SSL Certificate and Hosted Zone for the domain by following these [instructions](../guides/custom-domain.md#cleanup).
