# Install

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Prerequisites

1. [Docker](https://docs.docker.com/install)
2. [AWS credentials](aws-credentials.md)

## Spin up a cluster

See [cluster configuration](config.md) to learn how you can customize your cluster and [EC2 instances](ec2-instances.md) for an overview of several EC2 instance types.

<!-- CORTEX_VERSION_MINOR -->
```bash
# install the CLI on your machine
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/master/get-cli.sh)"

# provision infrastructure on AWS and spin up a cluster
cortex cluster up
```

Note: To use GPU nodes, you may need to subscribe to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM) and [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.

Note: Cortex creates and manages resources in your AWS account. The `cortex cluster up` command will display the cost of your cluster. Here is an example:

```text
aws resource                             cost per hour
1 eks cluster                            $0.10
1 - 4 m5.large instances for your apis   $0.0338 - $0.096 each (varies based on spot price)
1 - 4 20gb ebs volumes for your apis     $0.003 each
1 t3.medium instance for the operator    $0.0416
1 20gb ebs volume for the operator       $0.003
2 elastic load balancers                 $0.025 each
1 nat gateway                            $0.045

your cluster will cost $0.28 - $0.63 per hour based on the cluster size and spot instance availability
```

See [EC2 Instances](ec2-instances.md) for a list of some common EC2 instance types and their suggested use cases.

## Deploy a model

<!-- CORTEX_VERSION_MINOR -->

```bash
# clone the Cortex repository
git clone -b master https://github.com/cortexlabs/cortex.git

# navigate to the TensorFlow iris classification example
cd cortex/examples/tensorflow/iris-classifier

# deploy the model to the cluster
cortex deploy

# view the status of the api
cortex get --watch

# stream logs from the api
cortex logs iris-classifier

# get the api's endpoint
cortex get iris-classifier

# classify a sample
curl -X POST -H "Content-Type: application/json" \
  -d '{ "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 }' \
  <API endpoint>
```

## Cleanup

```bash
# delete the api
cortex delete iris-classifier
```

See [uninstall](uninstall.md) if you'd like to spin down your cluster.
