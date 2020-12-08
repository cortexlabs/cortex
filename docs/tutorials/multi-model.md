# Deploy a multi-model API

Deploy several models in a single API to improve resource utilization efficiency.

### Define a multi-model API

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
cx.create_api(api_spec, predictor=PythonPredictor, requirements=requirements)
```

### Deploy

```bash
$ python multi_model.py
```
