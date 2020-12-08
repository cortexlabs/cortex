# Deploy a project

## Install cortex

```bash
$ pip install cortex
```

## Create a directory

```bash
$ mkdir text-generator && cd text-generator

$ touch predictor.py requirements.txt realtime.py
```

## Define a Predictor

```python
# predictor.py

class PythonPredictor:
    def __init__(self, config):
        from transformers import pipeline

        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]
```

## Specify Python dependencies

```text
tensorflow
transformers
```

## Configure an API

```python
# realtime.py

import cortex

api_spec = {
    "name": "text-generator",
    "kind": "RealtimeAPI",
    "predictor": {"type": "python", "path": "predictor.py"},
}

cx = cortex.client("local")
cx.deploy(api_spec, project_dir=".")
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
$ curl http://localhost:8889 -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
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
$ curl https://***.execute-api.us-west-2.amazonaws.com/text-generator -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
```

## Delete the APIs

```bash
$ cortex delete --env local text-generator

$ cortex delete --env aws text-generator
```
