# Deploy machine learning models in production

Cortex is an open source platform that makes it simple to deploy your machine learning models as web APIs on AWS. 

Cortex combines TensorFlow Serving, ONNX Runtime, and Flask into a single tool that takes models from S3 and deploys them as web APIs. It also uses Docker and Kubernetes behind the scenes to autoscale, run rolling updates, and support CPU and GPU inference. The project is maintained by a venture-backed team of infrastructure engineers with backgrounds from Google, Illumio, and Berkeley.

<br>

<!-- CORTEX_VERSION_MINOR x2 (e.g. www.cortex.dev/v/0.8/...) -->
[install](https://www.cortex.dev/install) • [docs](https://www.cortex.dev) • [examples](examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [email us](mailto:hello@cortex.dev) • [chat with us](https://gitter.im/cortexlabs/cortex)

<br>

<!-- Set header Cache-Control=no-cache on the S3 object metadata (see https://help.github.com/en/articles/about-anonymized-image-urls) -->
![Demo](https://cortex-public.s3-us-west-2.amazonaws.com/demo/gif/v0.8.gif)

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

## How it works

### Define your deployment using declarative configuration

```yaml
# cortex.yaml

- kind: api
  name: my-api
  model: s3://my-bucket/my-model.onnx
  request_handler: handler.py
  compute:
    gpu: 1
```

### Customize request handling

```python
# handler.py

# Load data for preprocessing or postprocessing. For example:
labels = download_labels_from_s3()


def pre_inference(sample, metadata):
  # Python code


def post_inference(prediction, metadata):
  # Python code
```

### Deploy to AWS using the CLI

```bash
$ cortex deploy

Deploying ...
http://***.amazonaws.com/my-api  # Your API is ready!
```

### Serve real-time predictions via autoscaling JSON APIs running on AWS

```bash
$ curl http://***.amazonaws.com/my-api -d '{"a": 1, "b": 2, "c": 3}'

{ prediction: "def" }
```

<br>

## Installation

<!-- CORTEX_VERSION_README_MINOR -->

```bash
# Download the install script
$ curl -O https://raw.githubusercontent.com/cortexlabs/cortex/0.8/cortex.sh && chmod +x cortex.sh

# Install the Cortex CLI on your machine: the CLI sends configuration and code to the Cortex cluster
$ ./cortex.sh install cli

# Set your AWS credentials
$ export AWS_ACCESS_KEY_ID=***
$ export AWS_SECRET_ACCESS_KEY=***

# Configure AWS instance settings
$ export CORTEX_NODE_TYPE="m5.large"
$ export CORTEX_NODES_MIN="2"
$ export CORTEX_NODES_MAX="5"

# Install the Cortex cluster in your AWS account: the cluster is responsible for hosting your APIs
$ ./cortex.sh install
```

<!-- CORTEX_VERSION_MINOR (e.g. www.cortex.dev/v/0.8/...) -->
See [installation instructions](https://www.cortex.dev/cluster/install) for more details.

<br>

## Examples

<!-- CORTEX_VERSION_README_MINOR x3 -->
- [Text generation](https://github.com/cortexlabs/cortex/tree/0.8/examples/text-generator) with GPT-2

- [Sentiment analysis](https://github.com/cortexlabs/cortex/tree/0.8/examples/sentiment-analysis) with BERT

- [Image classification](https://github.com/cortexlabs/cortex/tree/0.8/examples/image-classifier) with Inception v3
