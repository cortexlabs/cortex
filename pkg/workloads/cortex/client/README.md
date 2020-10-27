Cortex makes it simple to deploy machine learning models in production.

### Deploy

* Deploy TensorFlow, PyTorch, ONNX, scikit-learn, and other models.
* Define preprocessing and postprocessing steps in Python.
* Configure APIs as realtime or batch.
* Deploy multiple models per API.

### Manage

* Monitor API performance and track predictions.
* Update APIs with no downtime.
* Stream logs from APIs.
* Perform A/B tests.

### Scale

* Test locally, scale on your AWS account.
* Autoscale to handle production traffic.
* Reduce cost with spot instances.

<!-- CORTEX_VERSION_MINOR -->
[documentation](https://docs.cortex.dev) • [tutorial](https://docs.cortex.dev/deployments/realtime-api/text-generator) • [examples](https://github.com/cortexlabs/cortex/tree/0.21/examples) • [chat with us](https://gitter.im/cortexlabs/cortex)

## Install the CLI

<!-- CORTEX_VERSION_MINOR -->
```bash
pip install cortex
```

You must have [Docker](https://docs.docker.com/install) installed to run Cortex locally or to create a cluster on AWS.

## Deploy an example

<!-- CORTEX_VERSION_MINOR -->
```bash
# clone the Cortex repository
git clone -b 0.21 https://github.com/cortexlabs/cortex.git

# navigate to the Pytorch text generator example
cd cortex/examples/pytorch/text-generator
```

### In Python
```python
import cortex
import requests

local_client = cortex.client("local")

# deploy the model as a realtime api and wait for it to become active
deployments = local_client.deploy("./cortex.yaml", wait=True)

# get the api's endpoint
url = deployments[0]["api"]["endpoint"]

# generate text
print(requests.post(url, json={"text": "machine learning is"}).text)

# delete the api
local_client.delete_api("text-generator")
```

### Using the CLI
```bash
# deploy the model as a realtime api
cortex deploy

# view the status of the api
cortex get --watch

# stream logs from the api
cortex logs text-generator

# get the api's endpoint
cortex get text-generator

# generate text
curl <API endpoint> \
  -X POST -H "Content-Type: application/json" \
  -d '{"text": "machine learning is"}'

# delete the api
cortex delete text-generator
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

You can now run the same commands shown above to deploy the text generator to AWS (if you didn't set the default CLI environment, add `--env aws` to the `cortex` commands).
