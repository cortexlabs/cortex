# Install

## AWS account and access key

As of now, Cortex only runs on AWS. We plan to support other cloud providers in the future. If you don't have an AWS account you can get started with one [here](https://portal.aws.amazon.com/billing/signup#/start).

Follow this [tutorial](https://aws.amazon.com/premiumsupport/knowledge-center/create-access-key) to create an access key.

**Note**

* Enable programmatic access for the IAM user.
* Attach the existing policy `AdministratorAccess` to your IAM user, or see [security](security.md) for a minimal access configuration.
* Each of the steps below requires different permissions. Please ensure you have the required permissions for each of the steps you need to run. The `AdministratorAccess` policy will work for all steps.

## Download the install script

<!-- CORTEX_VERSION_MINOR -->

```bash
# Download
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex-installer.sh

# Change permissions
chmod +x cortex-installer.sh

# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***
```

## Kubernetes

Cortex runs on Kubernetes. If you don't already have a Kubernetes cluster, [eksctl](https://eksctl.io) is a simple tool to create and manage one.

**We recommend a minimum cluster size of 2 [t3.medium](https://aws.amazon.com/ec2/instance-types) AWS instances. Cortex may not run successfully on clusters with less compute resources.**

```bash
# Install kubectl, eksctl, and aws-iam-authenticator
./cortex-installer.sh install kubernetes-tools

# Spin up an EKS cluster (this takes ~20 minutes; see eksctl.io for more options)
eksctl create cluster --name=cortex --nodes=2 --node-type=t3.medium
```

This cluster configuration will cost about $0.29 per hour in AWS fees.

## Install the Operator

The Cortex operator is a service that runs on Kubernetes, translates declarative configuration into workloads, and orchestrates those workloads on the cluster. Its installation is configurable. For a full list of configuration options please refer to the [operator config](config.md) documentation.

```bash
# Install the Cortex operator
./cortex-installer.sh install operator
```

## Install the CLI

The CLI runs on developer machines (e.g. your laptop) and communicates with the operator.

```bash
# Install the Cortex CLI
./cortex-installer.sh install cli

# Get the operator endpoint
./cortex-installer.sh get endpoints

# Configure the CLI
cortex configure
```
