# Deploy machine learning models in production

Cortex is an open source platform that takes machine learning models—trained with nearly any framework—and turns them into production web APIs in one command. <br>

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR x1 -->
[install](https://www.cortex.dev/install) • [docs](https://www.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.9/examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [email us](mailto:hello@cortex.dev) • [chat with us](https://gitter.im/cortexlabs/cortex)<br><br>

<!-- Set header Cache-Control=no-cache on the S3 object metadata (see https://help.github.com/en/articles/about-anonymized-image-urls) -->
![Demo](https://cortex-public.s3-us-west-2.amazonaws.com/demo/gif/v0.8.gif)<br>

<br>

## Quickstart

Below, we'll walk through how to use Cortex to deploy OpenAI's GPT-2 model as a service on AWS. You'll need to [install Cortex](https://www.cortex.dev/install) on your AWS account before getting started.

<br>

### Step 1: Configure your deployment

The configuration below will download the model from the `cortex-examples` S3 bucket and deploy it as a web service that can serve real-time predictions.

```yaml
# cortex.yaml

- kind: deployment
  name: text

- kind: api
  name: generator
  model: s3://cortex-examples/text-generator/gpt-2/124M
  request_handler: handler.py
```

<!-- CORTEX_VERSION_README_MINOR -->
You can run the code that generated the model [here](https://colab.research.google.com/github/cortexlabs/cortex/blob/0.9/examples/text-generator/gpt-2.ipynb).

<br>

### Step 2: Add request handling

The model requires encoded data for inference, but the API should accept strings of natural language as input. It should also decode the inference output.

```python
# handler.py

from encoder import get_encoder
encoder = get_encoder()


def pre_inference(sample, metadata):
    context = encoder.encode(sample["text"])
    return {"context": [context]}


def post_inference(prediction, metadata):
    response = prediction["sample"]
    return encoder.decode(response)
```

<br>

### Step 3: Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster.

```bash
$ cortex deploy

deployment started
```

You can track the status of a deployment using `cortex get`.

```bash
$ cortex get generator --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           8s            123ms

url: http://***.amazonaws.com/text/generator
```

Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

<br>

### Step 4: Serve real-time predictions

Once you have your endpoint, you can make requests:

```bash
$ curl http://***.amazonaws.com/text/generator \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "machine learning"}'

Machine learning, with more than one thousand researchers around the world today, are looking to create computer-driven machine learning algorithms that can also be applied to human and social problems, such as education, health care, employment, medicine, politics, or the environment...
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).

<br>

## How Cortex works

The CLI sends configuration and code to the cluster every time you run `cortex deploy`. Each model is loaded from S3 into a Docker container, along with any Python packages and request handling code. The model is exposed as a web service using Elastic Load Balancing (ELB), Flask, TensorFlow Serving, and ONNX Runtime. The containers are orchestrated on Elastic Kubernetes Service (EKS) while logs and metrics are streamed to CloudWatch.

<br>

## More examples

<!-- CORTEX_VERSION_README_MINOR x3 -->
- [Iris classification](https://github.com/cortexlabs/cortex/tree/0.9/examples/iris-classifier)

- [Sentiment analysis](https://github.com/cortexlabs/cortex/tree/0.9/examples/sentiment-analysis) with BERT

- [Image classification](https://github.com/cortexlabs/cortex/tree/0.9/examples/image-classifier) with Inception v3 and AlexNet

<br>

## Key features

- **Autoscaling:** Cortex automatically scales APIs to handle production workloads.

- **Multi framework:** Cortex supports TensorFlow, Keras, PyTorch, Scikit-learn, XGBoost, and more.

- **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **Log streaming:** Cortex streams logs from deployed models to your CLI.

- **Prediction monitoring:** Cortex monitors network metrics and tracks predictions.

- **Minimal configuration:** Deployments are defined in a single `cortex.yaml` file.
