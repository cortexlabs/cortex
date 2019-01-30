# Install

## AWS account

As of now, Cortex only runs on AWS. We plan to support other cloud providers in the near future. If you don't have an AWS account you can get started with one [here](https://portal.aws.amazon.com/billing/signup#/start).

## AWS CLI

Please make sure that you have the AWS CLI installed on your machine ([installation instructions](https://docs.aws.amazon.com/cli/latest/userguide/installing.html)).

## kubectl

Please make sure that you have kubectl installed and configured correctly ([installation instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl
)).

## Kubernetes

Cortex runs on Kubernetes. Please make sure you have a Kubernetes cluster running before installing Cortex. We support versions 1.10 and 1.11.

If you don't already have a Kubernetes cluster, we recommend using [eksctl](https://eksctl.io) to create and manage one.

```bash
# Spin up an EKS cluster (this takes ~20 minutes for a small cluster)
eksctl create cluster --name=<name> --nodes=<num-nodes> --node-type=<node-type>
```

Note: we recommend a minimum cluster size of 3 [t3.small](https://aws.amazon.com/ec2/instance-types) AWS instances. Cortex may not run successfully on clusters with less compute resources.

## Cortex

Confirm that your Kubernetes cluster is running and that you have access to it:

```bash
kubectl cluster-info
```

Install Cortex in your cluster:

<!-- CORTEX_VERSION_STABLE -->

```bash
# Download install script
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex.sh

# Change permissions
chmod +x cortex.sh

# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Install
./cortex.sh install
```

Cortex installation is configurable. For a full list of configuration options please refer to the [cluster config](config.md) documentation.

## Install the Cortex CLI

### Mac

<!-- CORTEX_VERSION_STABLE -->

```bash
# Download
curl -O https://s3-us-west-2.amazonaws.com/get-cortex/cortex-cli-master-mac.zip

# Unzip
unzip cortex-cli-master-mac.zip

# Change permissions
chmod +x cortex

# Move the binary
sudo mv cortex /usr/local/bin/cortex

# Cleanup
rm cortex-cli-master-mac.zip

# Add bash completion scripts and the cx alias
echo 'source <(cortex completion)' >> ~/.bash_profile
```

### Linux

<!-- CORTEX_VERSION_STABLE -->

```bash
# Download
curl -O https://s3-us-west-2.amazonaws.com/get-cortex/cortex-cli-master-linux.zip

# Unzip
unzip cortex-cli-master-linux.zip

# Change permissions
chmod +x cortex

# Move the binary
sudo mv cortex /usr/local/bin/cortex

# Add bash completion scripts and the cx alias
echo 'source <(cortex completion)' >> ~/.bashrc

# Cleanup
rm cortex-cli-master-linux.zip
```

## Configure the Cortex CLI

```bash
# Get the operator URL
./cortex.sh endpoints

# Configure the CLI
cortex configure

# Environment: dev

# Cortex operator URL: ***

# AWS Access Key ID: ***

# AWS Secret Access Key: ***
```
