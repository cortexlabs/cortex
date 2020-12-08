# Deploy a multi-model API

## Install cortex

```bash
$ pip install cortex
```

## Spin up a cluster on AWS (requires AWS credentials)

```bash
$ cortex cluster up
```

## Define a multi-model API

```python
# multi_model.py

import cortex

class PythonPredictor:
    def __init__(self, config):
        from transformers import pipeline

        self.analyzer = pipeline(task="sentiment-analysis", device=device)
        self.summarizer = pipeline(task="summarization", device=device)

    def predict(self, query_params, payload):
        model = query_params.get("model")

        if model == "sentiment":
            return self.analyzer(payload["text"])[0]
        elif model == "summarizer":
            return self.summarizer(payload["text"])[0]["summary_text"]

requirements = ["tensorflow", "transformers"]

api_spec = {"name": "multi-model", "kind": "RealtimeAPI"}

cx = cortex.client("aws")
cx.deploy(api_spec, predictor=PythonPredictor, requirements=requirements)
```

## Deploy to AWS

```bash
$ python multi_model.py
```

## Monitor

```bash
$ cortex get multi-model --env aws --watch
```

## Stream logs

```bash
$ cortex logs multi-model
```

## Make a request

```bash
$ curl https://***.execute-api.us-west-2.amazonaws.com/text-generator?model=sentiment -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
```

## Delete the API

```bash
$ cortex delete multi-model
```
