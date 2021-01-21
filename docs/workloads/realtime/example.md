# RealtimeAPI

Deploy realtime APIs that can respond to prediction requests on demand.

## Key features

* Request-based autoscaling
* Multi-model endpoints
* Server-side batching
* Metrics and log aggregation
* Rolling updates

## Install Cortex

```bash
$ pip install cortex
```

## Create a cluster on AWS

```yaml
# cluster.yaml

region: us-east-1
instance_type: g4dn.xlarge
min_instances: 1
max_instances: 3
spot: true
```

```bash
$ cortex cluster up --config cluster.yaml
```

## Deploy a realtime API

### Implement your API

```bash
$ mkdir text-generator && cd text-generator
$ touch predictor.py requirements.txt text_generator.yaml
```

```python
# predictor.py

from transformers import pipeline

class PythonPredictor:
    def __init__(self, config):
        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]
```

```python
# requirements.txt

transformers
torch
```

```yaml
# text_generator.yaml

- name: text-generator
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
  compute:
    gpu: 1
```

### Deploy

```bash
$ cortex deploy text_generator.yaml
```

### Monitor

```bash
$ cortex get text-generator --watch
```

### Stream logs

```bash
$ cortex logs text-generator
```

### Make a request

```bash
$ curl http://***.elb.us-west-2.amazonaws.com/text-generator -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
```

### Delete

```bash
$ cortex delete text-generator
```
