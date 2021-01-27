<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Run inference at scale

Cortex is an open source platform for production inference workloads.

<br>

## Architecture

* Kubernetes - container orchestration
* Istio - traffic management
* Prometheus - metric aggregation
* Fluent Bit - structured logging
* FastAPI - request handling
* TensorFlow Serving / ONNX Runtime - model serving

<br>

## Workloads

### Realtime APIs - respond to prediction requests on demand

* Multi-framework support
* Request-based autoscaling
* Server-side batching
* Multi-model caching
* Live model reloading
* Rolling updates
* Traffic splitting
* Metric aggregation
* Structured logging

### Batch APIs - run distributed inference on large datasets

* Multi-framework support
* Automatic retries
* Dead letter queues
* Scale to 0
* Metric aggregation
* Structured logging

<br>

## How it works

### Install on Kubernetes

```text
$ helm install cortex
```

### Implement a Predictor

```python
# predictor.py

from transformers import pipeline

class PythonPredictor:
    def __init__(self, config):
        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]
```

### Configure a realtime API

```yaml
# text_generator.yaml

- name: text-generator
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
  compute:
    gpu: 1
    mem: 8Gi
  autoscaling:
    min_replicas: 1
    max_replicas: 10
```

### Deploy

```bash
$ cortex deploy text_generator.yaml

# creating http://example.com/text-generator

```

### Serve prediction requests

```bash
$ curl http://example.com/text-generator -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
```

<br>

## Get started

* [Cortex Core](https://docs.cortex.dev/core) - free and open source community offering.
* [Cortex Cloud](https://docs.cortex.dev/cloud) - managed Cortex clusters on AWS and GCP.
