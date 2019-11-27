# Deploy machine learning models in production

Cortex is an open source platform for deploying machine learning models—trained with nearly any framework—as production web services.

<br>

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->
[install](https://www.cortex.dev/install) • [tutorial](https://www.cortex.dev/v/master/iris-classifier) • [docs](https://www.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.10/examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [email us](mailto:hello@cortex.dev) • [chat with us](https://gitter.im/cortexlabs/cortex)<br><br>

<!-- Set header Cache-Control=no-cache on the S3 object metadata (see https://help.github.com/en/articles/about-anonymized-image-urls) -->
![Demo](https://d1zqebknpdh033.cloudfront.net/demo/gif/v0.8.gif)

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

### Implement your predictor

```python
# predictor.py

model = download_model()

def predict(sample, metadata):
    return model.predict(sample["text"])
```

### Configure your deployment

```yaml
# cortex.yaml

- kind: deployment
  name: sentiment

- kind: api
  name: classifier
  predictor:
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

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           8s            24ms

class     count
positive  8
negative  4
```

<br>

## How it works

The CLI sends configuration and code to the cluster every time you run `cortex deploy`. Each model is loaded into a Docker container, along with any Python packages and request handling code. The model is exposed as a web service using Elastic Load Balancing (ELB), TensorFlow Serving, and ONNX Runtime. The containers are orchestrated on Elastic Kubernetes Service (EKS) while logs and metrics are streamed to CloudWatch.

<br>

## Examples

<!-- CORTEX_VERSION_README_MINOR x4 -->
- [Sentiment analysis](https://github.com/cortexlabs/cortex/tree/0.10/examples/tensorflow/sentiment-analysis) in TensorFlow with BERT
- [Image classification](https://github.com/cortexlabs/cortex/tree/0.10/examples/tensorflow/image-classifier) in TensorFlow with Inception
- [Text generation](https://github.com/cortexlabs/cortex/tree/0.10/examples/pytorch/text-generator) in PyTorch with DistilGPT2
- [Iris classification](https://github.com/cortexlabs/cortex/tree/0.10/examples/xgboost/iris-classifier) in XGBoost / ONNX
