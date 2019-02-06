# Install

## AWS account

As of now, Cortex only runs on AWS. We plan to support other cloud providers in the future. If you don't have an AWS account you can get started with one [here](https://portal.aws.amazon.com/billing/signup#/start).

## Install script

<!-- CORTEX_VERSION_MINOR -->

```bash
# Download
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex.sh

# Change permissions
chmod +x cortex.sh

# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***
```

## Kubernetes

Cortex runs on Kubernetes. Please make sure you have a Kubernetes cluster running before installing Cortex. We support versions 1.10 and 1.11.

**We recommend a minimum cluster size of 3 [t3.small](https://aws.amazon.com/ec2/instance-types) AWS instances. Cortex may not run successfully on clusters with less compute resources.**

If you don't already have a Kubernetes cluster, [eksctl](https://eksctl.io) is a simple tool to create and manage one:

```bash
# Install kubectl, eksctl, and aws-iam-authenticator
./cortex.sh install kubernetes-tools

# Spin up an EKS cluster (see eksctl.io for more configuration options)
eksctl create cluster --name=cortex --nodes=3 --node-type=t3.small  # this takes ~20 minutes
```

## Operator

The operator installation is configurable. For a full list of configuration options please refer to the [operator config](operator/config.md) documentation.

```bash
# Install the Cortex operator
./cortex.sh install operator
```

## CLI

```bash
# Install the Cortex CLI
./cortex.sh install cli
```
