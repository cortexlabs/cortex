# Deploy a realtime API

_WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.23.*, run `git checkout -b 0.23` or switch to the `0.23` branch on GitHub)_

## Install cortex

```bash
$ pip install cortex
```

## Define a realtime API

```python
# realtime.py

import cortex

class PythonPredictor:
    def __init__(self, config):
        from transformers import pipeline

        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]

requirements = ["tensorflow", "transformers"]

api_spec = {"name": "text-generator", "kind": "RealtimeAPI"}

cx = cortex.client("local")
cx.deploy(api_spec, predictor=PythonPredictor, requirements=requirements)
```

## Test locally (requires Docker)

```bash
$ python realtime.py
```

## Monitor

```bash
$ cortex get text-generator --watch
```

## Make a request

```bash
$ curl http://localhost:8889 \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "hello world"}'
```

## Stream logs

```bash
$ cortex logs text-generator
```

## Spin up a cluster on AWS (requires AWS credentials)

```bash
$ cortex cluster up
```

## Edit `realtime.py`

```python
# cx = cortex.client("local")
cx = cortex.client("aws")
```

## Deploy to AWS

```bash
$ python realtime.py
```

## Monitor

```bash
$ cortex get text-generator --env aws --watch
```

## Make a request

```bash
$ curl https:// \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "hello world"}'
```

## Delete the API

```bash
$ cortex delete --env local text-generator

$ cortex delete --env aws text-generator
```
