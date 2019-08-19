# Get Started

## Prerequisites

1. [AWS credentials](aws.md)
2. [Docker](https://docs.docker.com/install)

## Installing Cortex in your AWS account

<!-- CORTEX_VERSION_MINOR -->

```bash
# Download
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex.sh

# Change permissions
chmod +x cortex.sh

# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Provision infrastructure on AWS and install Cortex
./cortex.sh install
```

See [cluster configuration](config.md) to customize your installation.

## Installing and configuring the CLI

```bash
# Install the Cortex CLI on your machine
./cortex.sh install cli

# Get the operator endpoint
./cortex.sh info

# Configure the CLI
cortex configure
```

## Creating a deployment

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

# Get the API's endpoint
cortex get tensorflow

# Classify a sample
curl -X POST -H "Content-Type: application/json" \
     -d '{ "samples": [ { "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 } ] }' \
     <API endpoint>
```

## Cleanup

```bash
# Delete the deployment
cortex delete iris
```

See [uninstall](uninstall.md) if you'd like to uninstall Cortex.
