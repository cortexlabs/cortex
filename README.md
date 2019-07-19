<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='88'>

<br>

**Get started:** [Install](https://docs.cortex.dev/install) • [Tutorial](https://docs.cortex.dev/tutorial) • <!-- CORTEX_VERSION_MINOR_STABLE e.g. https://docs.cortex.dev/v/0.2/ -->[Docs](https://docs.cortex.dev) • <!-- CORTEX_VERSION_MINOR_STABLE -->[Examples](https://github.com/cortexlabs/cortex/tree/0.6/examples)

**Learn more:** [Website](https://cortex.dev) • [Blog](https://blog.cortex.dev) • [Subscribe](https://cortexlabs.us20.list-manage.com/subscribe?u=a1987373ab814f20961fd90b4&id=ae83491e1c) • [Contact](mailto:hello@cortex.dev)

<br>

Cortex is a machine learning model deployment platform that runs in your AWS account. You define deployments with simple declarative configuration and Cortex deploys your models as JSON APIs. It also handles autoscaling, rolling updates, log streaming, inference on CPUs or GPUs, and more.

Cortex is actively maintained by a venture-backed team of infrastructure engineers and [we're hiring](https://angel.co/cortex-labs-inc/jobs).

<br>

## How it works

**Define** your deployment using declarative configuration:

```yaml
# cortex.yaml

- kind: api
  name: my-api
  model: s3://my-bucket/my-model.zip
  request_handler: handler.py
  compute:
    min_replicas: 5
    max_replicas: 20
```

**Customize** request handling (optional):

```python
# handler.py

def pre_inference(sample, metadata):
  # Python code


def post_inference(prediction, metadata):
  # Python code
```

**Deploy** to your cloud infrastructure:

```bash
$ cortex deploy

Deploying ...
https://amazonaws.com/my-api  # Your API is ready!
```

**Serve** real time predictions via scalable JSON APIs:

```bash
$ curl -d '{"a": 1, "b": 2, "c": 3}' https://amazonaws.com/my-api

{ prediction: "def" }
```

<br>

## Spinning up a Cortex cluster on your AWS account

```bash
# Download the install script
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex.sh && chmod +x cortex.sh

# Set your AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Provision infrastructure on AWS and install Cortex
./cortex.sh install

# Install the Cortex CLI on your machine
./cortex.sh install cli
```

<br>

## Key features

- **Minimal declarative configuration:** Deployments can be defined in a single `cortex.yaml` file.

- **Autoscaling:** Cortex can automatically scale APIs to handle production workloads.

- **Multi framework:** Cortex supports TensorFlow, Keras, PyTorch, Scikit-learn, XGBoost, and more.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **Log streaming:** Cortex streams logs from your deployed models to your CLI.

- **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.
