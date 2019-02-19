# Uninstall

## Uninstall script

<!-- CORTEX_VERSION_MINOR -->

```bash
# Download
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex.sh

# Change permissions
chmod +x cortex.sh
```

## Operator

```bash
# Uninstall the Cortex operator
./cortex.sh uninstall operator
```

## CLI

```bash
# Uninstall the Cortex CLI
./cortex.sh uninstall cli
```

## AWS

```bash
# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Delete the S3 bucket
aws s3 ls
aws s3 rb --force s3://<bucket-name>

# Delete the log group
aws logs delete-log-group --log-group-name cortex --region us-west-2

# Uninstall the AWS CLI (if you used cortex.sh to install it)
sudo rm -rf /usr/local/aws && sudo rm /usr/local/bin/aws && rm -rf ~/.aws
```

## Kubernetes

If you used [`eksctl`](https://eksctl.io) to create your cluster, you can use it to spin the cluster down.

**Make sure the Cortex operator is uninstalled to prevent AWS resource deletion deadlocks.**

```bash
# Spin down an EKS cluster
eksctl delete cluster --name=cortex
# Confirm that both eksctl CloudFormation stacks have been deleted via the AWS console

# Uninstall kubectl, eksctl, and aws-iam-authenticator
./cortex.sh uninstall kubernetes-tools
```
