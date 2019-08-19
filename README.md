<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='88'>

<br>

**Get started:** [install](https://docs.cortex.dev/install) • [tutorial](https://docs.cortex.dev/tutorial) • <!-- CORTEX_VERSION_MINOR_STABLE e.g. https://docs.cortex.dev/v/0.2/ -->[docs](https://docs.cortex.dev) • <!-- CORTEX_VERSION_MINOR_STABLE -->[examples](https://github.com/cortexlabs/cortex/tree/0.7/examples)

**Learn more:** [website](https://cortex.dev) • [blog](https://medium.com/cortex-labs) • [subscribe](https://cortexlabs.us20.list-manage.com/subscribe?u=a1987373ab814f20961fd90b4&id=ae83491e1c) • [contact](mailto:hello@cortex.dev)

<br>

Cortex is a machine learning deployment platform that runs in your AWS account. It takes exported models from S3 and deploys them as web APIs. It also handles autoscaling, rolling updates, log streaming, inference on CPUs or GPUs, and more.

Cortex is maintained by a venture-backed team of infrastructure engineers and [we're hiring](https://angel.co/cortex-labs-inc/jobs).

<br>

## How it works

**Define** your deployment using declarative configuration:

```yaml
# cortex.yaml

- kind: api
  name: my-api
  model: s3://my-bucket/my-model.onnx
  request_handler: handler.py
```

**Customize** request handling (optional):

```python
# handler.py


# Load data for preprocessing or postprocessing. For example:
labels = download_my_labels_from_s3()


def pre_inference(sample, metadata):
  # Python code


def post_inference(prediction, metadata):
  # Python code
```

**Deploy** to AWS:

```bash
$ cortex deploy

Deploying ...
https://***.amazonaws.com/my-api  # Your API is ready!
```

**Serve** real time predictions via scalable JSON APIs:

```bash
$ curl -d '{"a": 1, "b": 2, "c": 3}' https://***.amazonaws.com/my-api

{ prediction: "def" }
```

<br>

## Hosting Cortex on AWS

```bash
# Download the install script
$ curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex.sh && chmod +x cortex.sh

# Set your AWS credentials
$ export AWS_ACCESS_KEY_ID=***
$ export AWS_SECRET_ACCESS_KEY=***

# Provision infrastructure on AWS and install Cortex
$ ./cortex.sh install

# Install the Cortex CLI on your machine
$ ./cortex.sh install cli
```

<br>

## Key features

- **Minimal declarative configuration:** Deployments can be defined in a single `cortex.yaml` file.

- **Autoscaling:** Cortex can automatically scale APIs to handle production workloads.

- **Multi framework:** Cortex supports TensorFlow, Keras, PyTorch, Scikit-learn, XGBoost, and more.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **Log streaming:** Cortex streams logs from your deployed models to your CLI.

- **Prediction Monitoring:** Cortex can monitor network metrics and track predictions.

- **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.
