# Install

## Prerequisites

1. [Docker](https://docs.docker.com/install)
2. [AWS credentials](aws.md)

## Installation

See [cluster configuration](config.md) to learn how you can customize your cluster.

<!-- CORTEX_VERSION_MINOR -->
```bash
# Install the Cortex CLI on your machine
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/0.11/get-cli.sh)"

# Provision infrastructure on AWS and install Cortex
cortex cluster up
```

Note: This will create resources in your AWS account which aren't included in the free tier, e.g. an EKS cluster, two Elastic Load Balancers, and EC2 instances (quantity and type as specified above). To use GPU nodes, you may need to subscribe to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM) and [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.

## Deploy a model

<!-- CORTEX_VERSION_MINOR -->

```bash
# Clone the Cortex repository
git clone -b 0.11 https://github.com/cortexlabs/cortex.git

# Navigate to the iris classifier example
cd cortex/examples/sklearn/iris-classifier

# Deploy the model to the cluster
cortex deploy

# View the status of the deployment
cortex get --watch

# Get the API's endpoint
cortex get classifier

# Classify a sample
curl -X POST -H "Content-Type: application/json" \
     -d '{ "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4 "petal_width": 0.3 }' \
     <API endpoint>
```

## Cleanup

```bash
# Delete the deployment
cortex delete iris
```

See [uninstall](uninstall.md) if you'd like to uninstall Cortex.
