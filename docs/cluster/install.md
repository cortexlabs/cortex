# Install

## Prerequisites

1. [AWS credentials](aws.md)
2. [Docker](https://docs.docker.com/install)

## Spin up Cortex in your AWS account

<!-- CORTEX_VERSION_MINOR -->

```bash
# Download
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex.sh

# Change permissions
chmod +x cortex.sh

# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Install Cortex
./cortex.sh install
```

## Install the CLI

```bash
# Install the Cortex CLI
./cortex.sh install cli

# Get the operator endpoint
./cortex.sh get endpoints

# Configure the CLI
cortex configure
```

## Create a deployment

<!-- CORTEX_VERSION_MINOR -->

```bash
# Clone the Cortex repository
git clone -b master https://github.com/cortexlabs/cortex.git

# Navigate to the iris classification example
cd cortex/examples/iris

# Deploy the model to the cluster
cortex deploy

# View the status of the deployment
cortex get --watch

# Classify a sample
cortex predict iris-type irises.json
```

## Cleanup

```bash
# Delete the deployment
cortex delete iris
```

See [uninstall](uninstall.md) if you'd like to uninstall Cortex.
