# Deploy machine learning models in production

Cortex is an open source platform that takes machine learning models—trained with nearly **any framework**—and turns them into **production web APIs** in one command. <br>

<!-- CORTEX_VERSION_MINOR x2 (e.g. www.cortex.dev/v/0.8/...) -->
[install](https://www.cortex.dev/install) • [docs](https://www.cortex.dev) • [examples](examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [email us](mailto:hello@cortex.dev) • [chat with us](https://gitter.im/cortexlabs/cortex)

<br>

<!-- Set header Cache-Control=no-cache on the S3 object metadata (see https://help.github.com/en/articles/about-anonymized-image-urls) -->
![Demo](https://cortex-public.s3-us-west-2.amazonaws.com/demo/gif/v0.8.gif)<br>

<br>

## Quickstart

<!-- CORTEX_VERSION_MINOR x2 (e.g. www.cortex.dev/v/0.8/...) -->
Below, we'll walk through how to use Cortex to deploy OpenAI's GPT-2 model as a service on AWS. You'll need to [install Cortex](https://www.cortex.dev/install) on your [AWS account](aws.com) before getting started.

<br>

### Step 1. Configure your deployment

Define a `deployment` and an `api` resource as shown below:

```yaml
# cortex.yaml

- kind: deployment
  name: text

- kind: api
  name: generator
  model: s3://cortex-examples/text-generator/gpt-2/124M
  request_handler: handler.py
```

<!-- CORTEX_VERSION_MINOR -->

A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes a model available as a web service that can serve real-time predictions. The above example configuration will download the model from the `cortex-examples` S3 bucket. You can run the code that generated the exported GPT-2 model [here](https://colab.research.google.com/github/cortexlabs/cortex/blob/master/examples/text-generator/gpt-2.ipynb).

<br>

### Step 2. Add request handling

The model requires encoded data for inference, but the API should accept strings of natural language as input. It should also decode the inference output. This can be implemented in a request handler file using the `pre_inference` and `post_inference` functions:

```python
# handler.py

from encoder import get_encoder
encoder = get_encoder()


def pre_inference(sample, metadata):
    context = encoder.encode(sample["text"])
    return {"context": [context]}


def post_inference(prediction, metadata):
    response = prediction["sample"]
    return {encoder.decode(response)}
```

<br>

### Step 3. Deploy to AWS

Deploying to AWS is as simple as running `cortex deploy` from your CLI. `cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes the model, makes it servable using TensorFlow Serving, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

You can track the status of a deployment using `cortex get`:

```bash
$ cortex get generator --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           8s            123ms

url: http://***.amazonaws.com/text/generator
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

<br>

### Step 4. Serve real-time predictions

Once you have your endpoint, you can make requests:

```bash
$ curl http://***.amazonaws.com/text/generator \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "machine learning"}'

Machine learning, with more than one thousand researchers around the world today, are looking to create computer-driven machine learning algorithms that can also be applied to human and social problems, such as education, health care, employment, medicine, politics, or the environment...
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).

<br>

## More Examples

<!-- CORTEX_VERSION_README_MINOR x3 -->
- [Iris classification](https://github.com/cortexlabs/cortex/tree/master/examples/iris-classifier)

- [Sentiment analysis](https://github.com/cortexlabs/cortex/tree/master/examples/sentiment-analysis) with BERT

- [Image classification](https://github.com/cortexlabs/cortex/tree/master/examples/image-classifier) with Inception v3

<br>

## Key features

- **Minimal declarative configuration:** Deployments can be defined in a single `cortex.yaml` file.

- **Autoscaling:** Cortex automatically scales APIs to handle production workloads.

- **Multi framework:** Cortex supports TensorFlow, Keras, PyTorch, Scikit-learn, XGBoost, and more.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **Log streaming:** Cortex streams logs from your deployed models to your CLI.

- **Prediction monitoring:** Cortex can monitor network metrics and track predictions.

- **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.
