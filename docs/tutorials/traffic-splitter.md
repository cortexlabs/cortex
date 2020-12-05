# Deploy a traffic splitter

_WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.23.*, run `git checkout -b 0.23` or switch to the `0.23` branch on GitHub)_

## Install cortex

```bash
$ pip install cortex
```

## Spin up a cluster on AWS (requires AWS credentials)

```bash
$ cortex cluster up
```

## Define 2 realtime APIs and a traffic splitter

```python
# traffic_splitter.py

import cortex

class PythonPredictor:
    def __init__(self, config):
        from transformers import pipeline

        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]

requirements = ["tensorflow", "transformers"]

api_spec_cpu = {
    "name": "text-generator-cpu",
    "kind": "RealtimeAPI",
    "compute": {
        "cpu": 1,
    },
}

api_spec_gpu = {
    "name": "text-generator-gpu",
    "kind": "RealtimeAPI",
    "compute": {
        "gpu": 1,
    },
}

traffic_splitter = {
    "name": "text-generator",
    "kind": "TrafficSplitter",
    "apis": [
        {"name": "text-generator-cpu", "weight": 30},
        {"name": "text-generator-gpu", "weight": 70},
    ],
}

cx = cortex.client("aws")
cx.deploy(api_spec_cpu, predictor=PythonPredictor, requirements=requirements)
cx.deploy(api_spec_gpu, predictor=PythonPredictor, requirements=requirements)
cx.deploy(traffic_splitter)
```

## Deploy to AWS

```bash
$ python traffic_splitter.py
```

## Monitor

```bash
$ cortex get text-generator --env aws --watch
```

## Stream logs

```bash
$ cortex logs text-generator
```

## Make a request

```bash
$ curl https:// \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "hello world"}'
```

## Delete the API

```bash
$ cortex delete text-generator
```
