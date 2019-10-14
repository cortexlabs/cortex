# Cortex
# Deploy machine learning models in production

Cortex is an open source platform that takes machine learning models—trained with nearly **any framework**—and turns them into **autoscaling**, **robust APIs**, in one command. <br><br>
<!-- CORTEX_VERSION_MINOR x2 (e.g. www.cortex.dev/v/0.8/...) -->
[install](https://www.cortex.dev/install) • [docs](https://www.cortex.dev) • [examples](examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [email us](mailto:hello@cortex.dev) • [chat with us](https://gitter.im/cortexlabs/cortex)

## How does Cortex work?

Cortex works by abstracting away the many disparate steps and technologies required to effectively deploy machine learning models in production. <br>
<br> 
Without a tool like Cortex, engineers need to wrangle tools like TensorFlow Serving, ONNX Runtime, Flask, Docker, AWS, and Kubernetes just to deploy a reliable API. Cortex spins up and manages all of those services with a single "cortex deploy" command.
<br>
<br>

<!-- Set header Cache-Control=no-cache on the S3 object metadata (see https://help.github.com/en/articles/about-anonymized-image-urls) -->
![Demo](https://cortex-public.s3-us-west-2.amazonaws.com/demo/gif/v0.8.gif)

<br>

## Cortex Quickstart

Below, we'll walkthrough how to use Cortex to deploy a classifier built on the famous iris data set.
<br><br>
What you'll need:

- An AWS account (you can set one up for free [here](aws.com))
- [Docker](https://www.docker.com/products/docker-desktop)
- Cortex installed on your machine ([see instructions here](https://www.cortex.dev/v/0.8/install))

### Step 1. Configure your deployment

The first step to deploying your model is to upload your model to an accessible S3 bucket. For this example, we have uploaded our model to our S3 bucket, located at s3://cortex-examples/iris-classifier/tensorflow <br><br>
Next, create a .yaml file titled `cortex.yaml`. This file will instruct Cortex as to how your deployment should be constructed. Within `cortex.yaml`, define a `deployment` and an `api` resource as shown below:

```yaml
- kind: deployment
  name: iris

- kind: api
  name: classifier
  model: s3://cortex-examples/iris-classifier/tensorflow
  request_handler: handlers/tensorflow.py
  tracker:
    model_type: classification
```

A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes a model available as a web service that can serve real-time predictions. The above example configuration will download the model from the `cortex-examples` S3 bucket.

<!-- CORTEX_VERSION_MINOR x5 -->
You can run the code that generated the exported models used in this folder example here:
- [TensorFlow](https://colab.research.google.com/github/cortexlabs/cortex/blob/0.8/examples/iris-classifier/models/tensorflow.ipynb)
- [Pytorch](https://colab.research.google.com/github/cortexlabs/cortex/blob/0.8/examples/iris-classifier/models/pytorch.ipynb)
- [Keras](https://colab.research.google.com/github/cortexlabs/cortex/blob/0.8/examples/iris-classifier/models/keras.ipynb)
- [XGBoost](https://colab.research.google.com/github/cortexlabs/cortex/blob/0.8/examples/iris-classifier/models/xgboost.ipynb)
- [sklearn](https://colab.research.google.com/github/cortexlabs/cortex/blob/0.8/examples/iris-classifier/models/sklearn.ipynb)

### Step 2. Add request handling

The API should convert the model’s prediction to a human readable label before responding to the client. This can be implemented in a request handler file, which we referenced in `cortex.yaml` as `handlers/tensorflow.py`. Your `handlers/tensorflow.py` should look like this:

```python
labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def post_inference(prediction, metadata):
    label_index = int(prediction["class_ids"][0])
    return labels[label_index]
```

### Step 3. Deploy to AWS

Deploying to AWS is as simple as running `cortex deploy` from your command line (from within the same folder as `cortex.yaml` and `handlers/tensorflow.py`). `cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes the model, makes it servable using TensorFlow Serving, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

You can track the status of a deployment using `cortex get`:

```bash
$ cortex get classifier --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           8s            -
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

### Step 4. Serve real-time predictions

```bash
$ cortex get classifier

url: http://***.amazonaws.com/iris/classifier

$ curl http://***.amazonaws.com/iris/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3}'

"iris-setosa"
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).

<br>

## More Examples

<!-- CORTEX_VERSION_README_MINOR x3 -->
- [Text generation](https://github.com/cortexlabs/cortex/tree/0.8/examples/text-generator) with GPT-2

- [Sentiment analysis](https://github.com/cortexlabs/cortex/tree/0.8/examples/sentiment-analysis) with BERT

- [Image classification](https://github.com/cortexlabs/cortex/tree/0.8/examples/image-classifier) with Inception v3

## What else can Cortex do?

Here's a slightly more detailed breakdown of Cortex's core features:

- **Minimal declarative configuration:** Deployments can be defined in a single `cortex.yaml` file.

- **Autoscaling:** Cortex automatically scales APIs to handle production workloads.

- **Multi framework:** Cortex supports TensorFlow, Keras, PyTorch, Scikit-learn, XGBoost, and more.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **Log streaming:** Cortex streams logs from your deployed models to your CLI.

- **Prediction monitoring:** Cortex can monitor network metrics and track predictions.

- **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.
