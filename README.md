<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

[Website](https://www.cortex.dev) • [Slack](https://community.cortex.dev) • [Docs](https://docs.cortex.dev)

<br>

# Model serving at scale

Cortex is a platform for deploying, managing, and scaling machine learning in production.

<br>

## Key features

* Run realtime inference, batch inference, and training workloads.
* Deploy TensorFlow, PyTorch, ONNX, and other models to production.
* Scale to handle production workloads with server-side batching and request-based autoscaling.
* Configure rolling updates and live model reloading to update APIs without downtime.
* Serve models efficiently with multi-model caching and spot / preemptible instances.
* Stream performance metrics and structured logs to any monitoring tool.
* Perform A/B tests with configurable traffic splitting.

<br>

## How it works

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
