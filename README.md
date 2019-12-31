# Deploy machine learning models in production

Cortex is an open source platform for deploying machine learning models as production web services.

<br>

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->
[install](https://www.cortex.dev/install) • [tutorial](https://www.cortex.dev/iris-classifier) • [docs](https://www.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.12/examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [email us](mailto:hello@cortex.dev) • [chat with us](https://gitter.im/cortexlabs/cortex)<br><br>

<!-- Set header Cache-Control=no-cache on the S3 object metadata (see https://help.github.com/en/articles/about-anonymized-image-urls) -->
![Demo](https://d1zqebknpdh033.cloudfront.net/demo/gif/v0.12.gif)

<br>

## Key features

* **Multi framework:** Cortex supports TensorFlow, PyTorch, scikit-learn, XGBoost, and more.
* **Autoscaling:** Cortex automatically scales APIs to handle production workloads.
* **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.
* **Spot instances:** Cortex supports EC2 spot instances.
* **Rolling updates:** Cortex updates deployed APIs without any downtime.
* **Log streaming:** Cortex streams logs from deployed models to your CLI.
* **Prediction monitoring:** Cortex monitors network metrics and tracks predictions.
* **Minimal configuration:** Cortex deployments are defined in a single `cortex.yaml` file.

<br>

## Spinning up a cluster

Cortex is designed to be self-hosted on any AWS account. You can spin up a cluster with a single command:

<!-- CORTEX_VERSION_README_MINOR -->
```bash
# install the CLI on your machine
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/0.12/get-cli.sh)"

# provision infrastructure on AWS and spin up a cluster
$ cortex cluster up

aws region: us-west-2
aws instance type: p2.xlarge
spot instances: yes
min instances: 0
max instances: 10

￮ spinning up your cluster ...
your cluster is ready!
```

<br>

## Deploying a model

### Implement your predictor

```python
# predictor.py

class PythonPredictor:
    def __init__(self, config):
        self.model = download_model()

    def predict(self, payload):
        return model.predict(payload["text"])
```

### Configure your deployment

```yaml
# cortex.yaml

- kind: deployment
  name: sentiment

- kind: api
  name: classifier
  predictor:
    type: python
    path: predictor.py
  tracker:
    model_type: classification
  compute:
    gpu: 1
    mem: 4G
```

### Deploy to AWS

```bash
$ cortex deploy

creating classifier (http://***.amazonaws.com/sentiment/classifier)
```

### Serve real-time predictions

```bash
$ curl http://***.amazonaws.com/sentiment/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "the movie was amazing!"}'

positive
```

### Monitor your deployment

```bash
$ cortex get classifier --watch

status   up-to-date   requested   last update   avg inference
live     1            1           8s            24ms

class     count
positive  8
negative  4
```

<br>

## What is Cortex similar to?

Cortex is an open source alternative to serving models with SageMaker or building your own model deployment platform on top of AWS services like Elastic Kubernetes Service (EKS), Elastic Container Service (ECS), Lambda, Fargate, and Elastic Compute Cloud (EC2) and open source projects like Docker, Kubernetes, and TensorFlow Serving.

<br>

## How does Cortex work?

The CLI sends configuration and code to the cluster every time you run `cortex deploy`. Each model is loaded into a Docker container, along with any Python packages and request handling code. The model is exposed as a web service using Elastic Load Balancing (ELB), TensorFlow Serving, and ONNX Runtime. The containers are orchestrated on Elastic Kubernetes Service (EKS) while logs and metrics are streamed to CloudWatch.

<br>

## Examples of Cortex deployments

<!-- CORTEX_VERSION_README_MINOR x5 -->
* [Sentiment analysis](https://github.com/cortexlabs/cortex/tree/0.12/examples/tensorflow/sentiment-analyzer): deploy a BERT model for sentiment analysis.
* [Image classification](https://github.com/cortexlabs/cortex/tree/0.12/examples/tensorflow/image-classifier): deploy an Inception model to classify images.
* [Search completion](https://github.com/cortexlabs/cortex/tree/0.12/examples/pytorch/search-completer): deploy Facebook's RoBERTa model to complete search terms.
* [Text generation](https://github.com/cortexlabs/cortex/tree/0.12/examples/pytorch/text-generator): deploy Hugging Face's DistilGPT2 model to generate text.
* [Iris classification](https://github.com/cortexlabs/cortex/tree/0.12/examples/sklearn/iris-classifier): deploy a scikit-learn model to classify iris flowers.
