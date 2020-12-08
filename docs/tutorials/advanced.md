# Advanced deployments

## Install cortex

```bash
$ pip install cortex
```

## Create a directory

```bash
$ mkdir text-generator && cd text-generator

$ touch predictor.py requirements.txt text-generator.yaml
```

## Define a Predictor in `predictor.py`

```python
class PythonPredictor:
    def __init__(self, config):
        from transformers import pipeline

        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]
```

## Specify Python dependencies in `requirements.txt`

```text
tensorflow
transformers
```

## Configure 2 realtime APIs and a traffic splitter in `text-generator.yaml`

```yaml
- name: text-generator-cpu
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
  compute:
    cpu: 1

- name: text-generator-gpu
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
  compute:
    gpu: 1

- name: text-generator
  kind: TrafficSplitter
  apis:
    - name: text-generator-cpu
      weight: 80
    - name: text-generator-gpu
      weight: 20
```

## Test locally (requires Docker)

```bash
$ cortex deploy text-generator.yaml
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

## Spin up a cluster on AWS

```bash
$ cortex cluster up
```

## Deploy to AWS

```bash
$ cortex deploy text-generator.yaml --env aws
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
$ cortex delete text-generator --env local

$ cortex delete text-generator --env aws
```
