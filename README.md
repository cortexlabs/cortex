<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Deploy machine learning models to production

Cortex is an open source platform for deploying, managing, and scaling machine learning in production.

<br>

## Model serving infrastructure

* Supports deploying TensorFlow, PyTorch, sklearn and other models as realtime or batch APIs.
* Ensures high availability with availability zones and automated instance restarts.
* Runs inference on spot instances with on-demand backups.
* Autoscales to handle production workloads.

#### Configure Cortex

```yaml
# cluster.yaml

region: us-east-1
instance_type: g4dn.xlarge
min_instances: 10
max_instances: 100
spot: true
```

#### Spin up Cortex on your AWS account

```text
$ cortex cluster up --config cluster.yaml

￮ configuring autoscaling ✓
￮ configuring networking ✓
￮ configuring logging ✓

cortex is ready!
```

<br>

## Reproducible deployments

* Package dependencies, code, and configuration for reproducible deployments.
* Configure compute, autoscaling, and networking for each API.
* Integrate with your data science platform or CI/CD system.
* Test locally before deploying to your cluster.

#### Implement a predictor

```python
from transformers import pipeline

class PythonPredictor:
  def __init__(self, config):
    self.model = pipeline(task="text-generation")

  def predict(self, payload):
    return self.model(payload["text"])[0]
```

#### Configure an API

```python
api_spec = {
  "name": "text-generator",
  "kind": "RealtimeAPI",
  "compute": {
    "gpu": 1,
    "mem": "8Gi"
  },
  "autoscaling": {
    "min_replicas": 1,
    "max_replicas": 10
  }
}
```

<br>

## Scalable machine learning APIs

* Scale to handle production workloads with request-based autoscaling.
* Stream performance metrics and logs to any monitoring tool.
* Serve many models efficiently with multi model caching.
* Configure traffic splitting for A/B testing.
* Update APIs without downtime.

#### Deploy to your cluster

```python
import cortex

cx = cortex.client("aws")
cx.create_api(api_spec, predictor=PythonPredictor)

# creating https://example.com/text-generator
```

#### Consume your API

```bash
$ curl https://example.com/text-generator -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
```

<br>

## Get started

```bash
pip install cortex
```

[Deploy models](https://docs.cortex.dev) and [join our community](https://gitter.im/cortexlabs/cortex).
