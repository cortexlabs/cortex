# Deploy machine learning models in production

Cortex is an open source platform that takes machine learning models—trained with nearly any framework—and turns them into production web APIs in one command. <br>

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->
[install](https://www.cortex.dev/install) • [docs](https://www.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.10/examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [email us](mailto:hello@cortex.dev) • [chat with us](https://gitter.im/cortexlabs/cortex)<br><br>

<!-- Set header Cache-Control=no-cache on the S3 object metadata (see https://help.github.com/en/articles/about-anonymized-image-urls) -->
![Demo](https://cortex-public.s3-us-west-2.amazonaws.com/demo/gif/v0.8.gif)

<br>

## Key features

- **Autoscaling:** Cortex automatically scales APIs to handle production workloads.

- **Multi framework:** Cortex supports TensorFlow, PyTorch, scikit-learn, XGBoost, and more.

- **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **Log streaming:** Cortex streams logs from deployed models to your CLI.

- **Prediction monitoring:** Cortex monitors network metrics and tracks predictions.

- **Minimal configuration:** Deployments are defined in a single `cortex.yaml` file.

<br>

## Usage

### Step 1: define your API

```python
# predictor.py

model = download_my_model()

def predict(sample, metadata):
    return model.predict(sample["text"])
```

### Step 2: configure your deployment

```yaml
# cortex.yaml

- kind: deployment
  name: sentiment

- kind: api
  name: classifier
  predictor:
    path: predictor.py
  compute:
    gpu: 1
```

### Step 3: deploy to AWS

```bash
$ cortex deploy

deployment started


$ cortex get classifier --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           8s            123ms

url: http://***.amazonaws.com/sentiment/classifier
```

### Step 4: serve real-time predictions

```bash
$ curl http://***.amazonaws.com/sentiment/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "the movie was great!"}'

positive
```

<br>

## How Cortex works

The CLI sends configuration and code to the cluster every time you run `cortex deploy`. Each model is loaded into a Docker container, along with any Python packages and request handling code. The model is exposed as a web service using Elastic Load Balancing (ELB), Flask, TensorFlow Serving, and ONNX Runtime. The containers are orchestrated on Elastic Kubernetes Service (EKS) while logs and metrics are streamed to CloudWatch.

<br>

## More examples

<!-- CORTEX_VERSION_README_MINOR x4 -->
- [Sentiment analysis](https://github.com/cortexlabs/cortex/tree/0.10/examples/tensorflow/sentiment-analysis) in TensorFlow with BERT
- [Image classification](https://github.com/cortexlabs/cortex/tree/0.10/examples/tensorflow/image-classifier) in TensorFlow with Inception
- [Text generation](https://github.com/cortexlabs/cortex/tree/0.10/examples/pytorch/text-generator) in PyTorch with DistilGPT2
- [Iris classification](https://github.com/cortexlabs/cortex/tree/0.10/examples/xgboost/iris-classifier) in XGBoost / ONNX
