# Uninstall

## Cortex

```bash
# Download install script
curl -O https://s3-us-west-2.amazonaws.com/get-cortex/cortex.sh
# Change permissions
chmod +x cortex.sh

# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Install
./cortex.sh uninstall
```

## Cortex CLI

```bash
sudo rm /usr/local/bin/cortex
```

## Delete AWS data

```bash
# Delete the S3 bucket
aws s3 rb s3://<bucket-name> --force

# Delete the log group
aws logs delete-log-group --log-group-name <log-group-name>
```

##Kubernetes

If you used [eksctl](https://eksctl.io) to create your cluster, you can use it to spin the cluster down.

```bash
# Spin down an EKS cluster
eksctl delete cluster --name=<name> [--region=<region>]
```
