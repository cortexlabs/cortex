# Deploy machine learning models in production

Cortex is an open source platform that takes machine learning models—trained with nearly any framework—and turns them into production web APIs in one command. <br><br>
<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR x1 -->
[quickstart](https://github.com/cortexlabs/cortex/blob/master/README.md#quickstart) • [how it works](https://github.com/cortexlabs/cortex/blob/master/README.md#how-cortex-works) • [docs](https://www.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.9/examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [email us](mailto:hello@cortex.dev) • [chat with us](https://gitter.im/cortexlabs/cortex)<br><br>

<!-- Set header Cache-Control=no-cache on the S3 object metadata (see https://help.github.com/en/articles/about-anonymized-image-urls) -->
![Demo](https://cortex-public.s3-us-west-2.amazonaws.com/demo/gif/v0.8.gif)<br>

<br>

## Quickstart

Below, we'll walk through how to use Cortex to deploy OpenAI's GPT-2 model as a service on AWS. You'll need to [install Cortex](https://www.cortex.dev/install) on your AWS account before getting started.

<br>

### Step 1: Configure your deployment

<!-- CORTEX_VERSION_README_MINOR -->
The configuration below will download the model from the `cortex-examples` S3 bucket.

```yaml
# cortex.yaml

- kind: deployment
  name: text

- kind: api
  name: generator
  model: s3://cortex-examples/text-generator/gpt-2/124M
  request_handler: handler.py
```
 You can run the code that generated the model [here](https://colab.research.google.com/github/cortexlabs/cortex/blob/0.9/examples/text-generator/gpt-2.ipynb).
<br>

### Step 2: Add request handling

The API should accept strings of natural language as input. It should also decode the inference output. This can be implemented in a request handler file using the `pre_inference` and `post_inference` functions:

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

Deploying to AWS is as simple as running `cortex deploy` from your CLI. `cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster. 

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

## More examples

<!-- CORTEX_VERSION_README_MINOR x3 -->
- [Iris classification](https://github.com/cortexlabs/cortex/tree/0.9/examples/iris-classifier)

- [Sentiment analysis](https://github.com/cortexlabs/cortex/tree/0.9/examples/sentiment-analysis) with BERT

- [Image classification](https://github.com/cortexlabs/cortex/tree/0.9/examples/image-classifier) with Inception v3 and AlexNet

<br>

## How Cortex works

Under the hood, Cortex uses a variety of tools to deploy your models. When you run `cortex deploy`, the following all happens automatically, without any manual input from you:

- Models are served using a combination of TensorFlow Serving, ONNX Runtime, and Flask to create an accessible API.

- Each individual model is containerized using Docker, which enables autoscaling, simplifies resource scheduling, and streamlines logging.

- A Kubernetes cluster is spun up to manage your Docker containers, making things like autoscaling and rolling updates possible. The cluster is managed using Amazon’s Elastic Kubernetes Service (EKS).

Again, all of these tools are launched and managed by Cortex. None of them require direct interface with you, the user. By tapping into these tools, Cortex is able to deliver features like:

- **Autoscaling:** Cortex automatically scales APIs to handle production workloads.

- **Multi framework:** Cortex supports TensorFlow, Keras, PyTorch, Scikit-learn, XGBoost, and more.

- **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **Log streaming:** Cortex streams logs from deployed models to your CLI.

- **Prediction monitoring:** Cortex monitors network metrics and tracks predictions.

- **Minimal declarative configuration:** Deployments are defined in a single `cortex.yaml` file.

