<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='88'>

<!-- CORTEX_VERSION_MINOR_STABLE x2 -->
[install](https://docs.cortex.dev/v/master/install) • [docs](https://docs.cortex.dev/v/master) • [examples](examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [email us](mailto:hello@cortex.dev) • [chat with us](https://gitter.im/cortexlabs/cortex)

<br>

![Cortex demo](https://s3-us-west-2.amazonaws.com/cortex-public/5fps.gif)

<br>

Cortex is a machine learning deployment platform that you can self-host on AWS. It combines TensorFlow Serving, ONNX Runtime, and Flask into a single tool that takes models from S3 and deploys them as REST APIs. It also uses Docker and Kubernetes behind the scenes to autoscale, run rolling updates, and support CPU and GPU inference.

<br>

## How it works

**Define your deployment using declarative configuration**

```yaml
# cortex.yaml

- kind: api
  name: my-api
  model: s3://my-bucket/my-model.onnx
  request_handler: handler.py
  compute:
    gpu: 1
```

**Customize request handling**

```python
# handler.py

# Load data for preprocessing or postprocessing. For example:
labels = download_labels_from_s3()


def pre_inference(sample, metadata):
  # Python code


def post_inference(prediction, metadata):
  # Python code
```

**Deploy to AWS using the CLI**

```bash
$ cortex deploy

Deploying ...
http://***.amazonaws.com/my-api  # Your API is ready!
```

**Serve real-time predictions via autoscaling JSON APIs running on AWS**

```bash
$ curl http://***.amazonaws.com/my-api -d '{"a": 1, "b": 2, "c": 3}'

{ prediction: "def" }
```

<br>

## Spinning up a Cortex cluster on AWS

<!-- CORTEX_VERSION_MINOR_STABLE -->

```bash
# Download the install script
$ curl -O https://raw.githubusercontent.com/cortexlabs/cortex/0.7/cortex.sh && chmod +x cortex.sh

# Install the Cortex CLI on your machine: the CLI sends configuration and code to the Cortex cluster
$ ./cortex.sh install cli

# Set your AWS credentials
$ export AWS_ACCESS_KEY_ID=***
$ export AWS_SECRET_ACCESS_KEY=***

# Configure AWS instance settings
$ export CORTEX_NODE_TYPE="p2.xlarge"
$ export CORTEX_NODES_MIN="1"
$ export CORTEX_NODES_MAX="3"

# Install the Cortex cluster in your AWS account: the cluster is responsible for hosting your APIs
$ ./cortex.sh install
```

<!-- CORTEX_VERSION_MINOR_STABLE -->
See [installation instructions](https://docs.cortex.dev/v/master/cluster/install) for more details.

<br>

## Key features

- **Minimal declarative configuration:** Deployments can be defined in a single `cortex.yaml` file.

- **Autoscaling:** Cortex automatically scales APIs to handle production workloads.

- **Multi framework:** Cortex supports TensorFlow, Keras, PyTorch, Scikit-learn, XGBoost, and more.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **Log streaming:** Cortex streams logs from your deployed models to your CLI.

- **Prediction monitoring:** Cortex can monitor network metrics and track predictions.

- **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.

<br>

## Examples

<!-- CORTEX_VERSION_MINOR_STABLE -->
- [Text generation](https://github.com/cortexlabs/cortex/tree/master/examples/text-generator) with GPT-2

<!-- CORTEX_VERSION_MINOR_STABLE -->
- [Sentiment analysis](https://github.com/cortexlabs/cortex/tree/master/examples/sentiment-analysis) with BERT

<!-- CORTEX_VERSION_MINOR_STABLE -->
- [Image classification](https://github.com/cortexlabs/cortex/tree/master/examples/image-classifier) with Inception v3
