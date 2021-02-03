<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Run inference at scale

Cortex is an open source platform for large-scale machine learning inference workloads.

<br>

## Workloads

### Realtime APIs - respond to prediction requests in real-time

* Deploy TensorFlow, PyTorch, and other models.
* Scale to handle production workloads with server-side batching and request-based autoscaling.
* Configure rolling updates and live model reloading to update APIs without downtime.
* Serve many models efficiently with multi-model caching.
* Perform A/B tests with configurable traffic splitting.
* Stream performance metrics and structured logs to any monitoring tool.

### Batch APIs - run distributed inference on large datasets

* Deploy TensorFlow, PyTorch, and other models.
* Configure the number of workers and the compute resources for each worker.
* Recover from failures with automatic retries and dead letter queues.
* Stream performance metrics and structured logs to any monitoring tool.

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

<br>

## Get started

* [Install Cortex](https://docs.cortex.dev)
* [Join our community](https://join.slack.com/t/cortex-dot-dev/shared_invite/zt-lf58axgy-0QkLZzFSSku5_Jybd9yiZQ)
