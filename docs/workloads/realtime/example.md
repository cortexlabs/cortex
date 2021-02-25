# RealtimeAPI

Create APIs that can respond to prediction requests in real-time.

## Implement

```bash
mkdir text-generator && cd text-generator
touch predictor.py requirements.txt text_generator.yaml
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

## Deploy

```bash
cortex deploy text_generator.yaml
```

## Monitor

```bash
cortex get text-generator --watch
```

## Stream logs

```bash
cortex logs text-generator
```

## Make a request

```bash
curl http://***.elb.us-west-2.amazonaws.com/text-generator -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
```

## Delete

```bash
cortex delete text-generator
```
