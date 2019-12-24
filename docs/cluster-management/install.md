# Install

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Prerequisites

1. [Docker](https://docs.docker.com/install)
2. [AWS credentials](aws-credentials.md)

## Installation

See [cluster configuration](config.md) to learn how you can customize your installation and [EC2 instances](ec2-instances.md) for an overview of how to pick an appropriate EC2 instance type for your cluster.

<!-- CORTEX_VERSION_MINOR -->
```bash
# install the CLI on your machine
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/master/get-cli.sh)"

# provision infrastructure on AWS and spin up a cluster
cortex cluster up
```

Note: This will create resources in your AWS account which aren't included in the free tier, e.g. an EKS cluster, two Elastic Load Balancers, and EC2 instances (quantity and type as specified above). To use GPU nodes, you may need to subscribe to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM) and [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.

## Deploy a model

<!-- CORTEX_VERSION_MINOR -->

```bash
# clone the Cortex repository
git clone -b master https://github.com/cortexlabs/cortex.git

# navigate to the iris classifier example
cd cortex/examples/sklearn/iris-classifier

# deploy the model to the cluster
cortex deploy

# view the status of the deployment
cortex get --watch

# stream logs from the API
cortex logs classifier

# get the API's endpoint
cortex get classifier

# classify a sample
curl -X POST -H "Content-Type: application/json" \
  -d '{ "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 }' \
  <API endpoint>
```

## Cleanup

```bash
# delete the deployment
cortex delete iris
```

See [uninstall](uninstall.md) if you'd like to uninstall Cortex.
