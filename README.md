# Deploy machine learning models in production

<br>

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->
[install](https://cortex.dev/install) • [docs](https://cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.15/examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [chat with us](https://gitter.im/cortexlabs/cortex)<br><br>

<!-- Set header Cache-Control=no-cache on the S3 object metadata (see https://help.github.com/en/articles/about-anonymized-image-urls) -->
![Demo](https://d1zqebknpdh033.cloudfront.net/demo/gif/v0.13_2.gif)

<br>

## Key features

* **Multi framework:** deploy TensorFlow, PyTorch, scikit-learn, and other models.
* **Autoscaling:** automatically scale APIs to handle production workloads.
* **CPU / GPU support:** run inference on CPU or GPU instances.
* **Spot instances:** save money with spot instances.
* **Rolling updates:** update deployed APIs with no downtime.
* **Log streaming:** stream logs from deployed models to your CLI.
* **Prediction monitoring:** monitor API performance and track predictions.

<br>

## Deploying a model

### Implement your predictor

```python
# predictor.py

class PythonPredictor:
    def __init__(self, config):
        self.model = download_model()

    def predict(self, payload):
        return self.model.predict(payload["text"])
```

### Configure your deployment

```yaml
# cortex.yaml

- name: sentiment-classifier
  predictor:
    type: python
    path: predictor.py
  compute:
    gpu: 1
    mem: 4G
```

### Deploy your model

```bash
$ cortex deploy

creating sentiment-classifier
```

### Serve predictions

```bash
$ curl http://localhost:12345 \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "serving models locally is cool!"}'

positive
```

<br>

## Running Cortex in production

### Spin up a cluster

Cortex clusters are designed to be self-hosted on any AWS account (GCP support is coming soon):

<!-- CORTEX_VERSION_README_MINOR -->
```bash
# install the CLI on your machine
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/0.15/get-cli.sh)"

# provision infrastructure on AWS and spin up a cluster
$ cortex cluster up

aws region: us-west-2
aws instance type: g4dn.xlarge
spot instances: yes
min instances: 0
max instances: 5

your cluster will cost $0.19 - $2.85 per hour based on cluster size and spot instance pricing/availability

￮ spinning up your cluster ...

your cluster is ready!
```

### Deploy to your cluster with the same code and configuration

```bash
$ cortex deploy

creating sentiment-classifier
```

### Serve predictions at scale

```bash
$ curl http://***.amazonaws.com/sentiment-classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "serving models at scale is really cool!"}'

positive
```

### Monitor your deployment

```bash
$ cortex get sentiment-classifier

status   up-to-date   requested   last update   avg request   2XX
live     1            1           8s            24ms          12

class     count
positive  8
negative  4
```

### How it works

The CLI sends configuration and code to the cluster every time you run `cortex deploy`. Each model is loaded into a Docker container, along with any Python packages and request handling code. The model is exposed as a web service using Elastic Load Balancing (ELB), TensorFlow Serving, and ONNX Runtime. The containers are orchestrated on Elastic Kubernetes Service (EKS) while logs and metrics are streamed to CloudWatch.

Cortex manages its own Kubernetes cluster so that end-to-end functionality like request-based autoscaling, GPU support, and spot instance management can work out of the box without any additional DevOps work.

<br>

## What is Cortex similar to?

Cortex is an open source alternative to serving models with SageMaker or building your own model deployment platform on top of AWS services like Elastic Kubernetes Service (EKS), Lambda, or Fargate and open source projects like Docker, Kubernetes, TensorFlow Serving, and TorchServe.

<br>

## Examples of Cortex deployments

<!-- CORTEX_VERSION_README_MINOR x3 -->
* [Image classification](https://github.com/cortexlabs/cortex/tree/0.15/examples/tensorflow/image-classifier): deploy an Inception model to classify images.
* [Search completion](https://github.com/cortexlabs/cortex/tree/0.15/examples/pytorch/search-completer): deploy Facebook's RoBERTa model to complete search terms.
* [Text generation](https://github.com/cortexlabs/cortex/tree/0.15/examples/pytorch/text-generator): deploy Hugging Face's DistilGPT2 model to generate text.
