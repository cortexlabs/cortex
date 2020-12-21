# RealtimeAPI

Deploy realtime APIs that can respond to prediction requests on demand.

## Key features

* Request-based autoscaling
* Multi-model endpoints
* Server-side batching
* Metrics and log aggregation
* Rolling updates

## How it works

### Install cortex

```bash
$ pip install cortex
```

### Spin up a cluster on AWS

```bash
$ cortex cluster up
```

### Define a realtime API

```python
# text_generator.py

import cortex

class PythonPredictor:
    def __init__(self, config):
        from transformers import pipeline
        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]

requirements = ["tensorflow", "transformers"]

api_spec = {"name": "text-generator", "kind": "RealtimeAPI"}

cx = cortex.client("aws")
cx.create_api(api_spec, predictor=PythonPredictor, requirements=requirements)
```

### Deploy to AWS

```bash
$ python text_generator.py
```

### Monitor

```bash
$ cortex get text-generator --env aws --watch
```

### Stream logs

```bash
$ cortex logs text-generator
```

### Make a request

```bash
$ curl https://***.execute-api.us-west-2.amazonaws.com/text-generator -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
```

### Delete the API

```bash
$ cortex delete text-generator --env aws
```
