# Install

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Install the CLI

<!-- CORTEX_VERSION_MINOR -->
```bash
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/master/get-cli.sh)"
```

You must also have [Docker](https://docs.docker.com/install) installed to run Cortex locally or to create a cluster on AWS.

## Deploy an example

<!-- CORTEX_VERSION_MINOR -->
```bash
# clone the Cortex repository
git clone -b master https://github.com/cortexlabs/cortex.git

# navigate to the TensorFlow iris classification example
cd cortex/examples/tensorflow/iris-classifier

# deploy the model
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

# delete the api
cortex delete iris-classifier
```

## Running at scale on AWS

Run the command below to create a cluster with basic configuration, or see [cluster configuration](config.md) to learn how you can customize your cluster with `cluster.yaml`.

See [EC2 instances](ec2-instances.md) for an overview of several EC2 instance types. To use GPU nodes, you may need to subscribe to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM) and [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.

```bash
# create a Cortex cluster on your AWS account
cortex cluster up

# set the default CLI environment (optional)
cortex env default aws
```

You can now run the same commands shown above to deploy the iris classifier to AWS (if you didn't set the default CLI environment, add `--env aws` to the `cortex` commands).

## Next steps

<!-- CORTEX_VERSION_MINOR -->
* Try the [tutorial](../../examples/sklearn/iris-classifier/README.md) to learn more about how to use Cortex.
* Deploy one of our [examples](https://github.com/cortexlabs/cortex/tree/master/examples).
* See our [exporting docs](../deployments/exporting.md) for how to export your model to use in an API.
* See [uninstall](uninstall.md) if you'd like to spin down your cluster.
