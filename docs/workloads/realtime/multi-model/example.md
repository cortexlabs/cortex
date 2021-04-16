# Multi-model API

Deploy several models in a single API to improve resource utilization efficiency.

## Define a multi-model API

```python
# multi_model.py

import cortex

class PythonPredictor:
    def __init__(self, config):
        from transformers import pipeline
        self.analyzer = pipeline(task="sentiment-analysis")

        import wget
        import fasttext
        wget.download(
            "https://dl.fbaipublicfiles.com/fasttext/supervised-models/lid.176.bin", "/tmp/model"
        )
        self.language_identifier = fasttext.load_model("/tmp/model")

    def predict(self, query_params, payload):
        model = query_params.get("model")
        if model == "sentiment":
            return self.analyzer(payload["text"])[0]
        elif model == "language":
            return self.language_identifier.predict(payload["text"])[0][0][-2:]

requirements = ["tensorflow", "transformers", "wget", "fasttext"]

api_spec = {"name": "multi-model", "kind": "RealtimeAPI"}

cx = cortex.client("aws")
cx.create_api(api_spec, predictor=PythonPredictor, requirements=requirements)
```

## Deploy

```bash
python multi_model.py
```
