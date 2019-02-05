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

## Delete AWS data

```bash
# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Delete the S3 bucket
aws s3 rb s3://<bucket-name> --force

# Delete the log group
aws logs delete-log-group --log-group-name <log-group-name>
```

## Kubernetes

If you used [`eksctl`](https://eksctl.io) to create your cluster, you can use it to spin the cluster down.

```bash
# Spin down an EKS cluster
eksctl delete cluster --name=<name>

# Install kubectl, eksctl, and aws-iam-authenticator
./cortex.sh uninstall kubernetes-tools
```
