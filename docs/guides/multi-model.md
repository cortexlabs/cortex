# Multi-model endpoints

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

It is possible to serve multiple models in the same Cortex API when using the Python predictor type (support for the TensorFlow predictor type is [coming soon](https://github.com/cortexlabs/cortex/issues/890)). In this guide, we'll deploy a sentiment analyzer and a text summarizer in one API, and we'll use query parameters to select the model. We'll be sharing a single GPU across both models.

## Step 1: implement your API

Create a new folder called "multi-model", and add these files:

### `cortex.yaml`

```yaml
- name: text-analyzer
  predictor:
    type: python
    path: predictor.py
  compute:
    cpu: 1
    gpu: 1
    mem: 12G
  autoscaling:
    threads_per_worker: 1
```

_Note: `threads_per_worker: 1` is the default, but setting it higher in production may increase throughput, especially when inference on one model takes a lot longer than the other(s)._

### `requirements.txt`

```python
torch
transformers==2.9.*
```

### `predictor.py`

```python
import torch
from transformers import pipeline


class PythonPredictor:
    def __init__(self, config):
        device = 0 if torch.cuda.is_available() else -1
        print(f"using device: {'cuda' if device == 0 else 'cpu'}")

        self.analyzer = pipeline(task="sentiment-analysis", device=device)
        self.summarizer = pipeline(task="summarization", device=device)

    def predict(self, query_params, payload):
        if query_params.get("model") == "sentiment":
            return self.analyzer(payload["text"])[0]
        elif query_params.get("model") == "summarizer":
            summary = self.summarizer(payload["text"])
            return summary[0]["summary_text"]
        else:
            return {"error": f"unknown model: {query_params.get('model')}"}
```

### sample-sentiment.json

```json
{
  "text": "best day ever"
}
```

### sample-summarizer.json

```json
{
  "text": "Machine learning (ML) is the scientific study of algorithms and statistical models that computer systems use to perform a specific task without using explicit instructions, relying on patterns and inference instead. It is seen as a subset of artificial intelligence. Machine learning algorithms build a mathematical model based on sample data, known as training data, in order to make predictions or decisions without being explicitly programmed to perform the task. Machine learning algorithms are used in a wide variety of applications, such as email filtering and computer vision, where it is difficult or infeasible to develop a conventional algorithm for effectively performing the task. Machine learning is closely related to computational statistics, which focuses on making predictions using computers. The study of mathematical optimization delivers methods, theory and application domains to the field of machine learning. Data mining is a field of study within machine learning, and focuses on exploratory data analysis through unsupervised learning. In its application across business problems, machine learning is also referred to as predictive analytics."
}
```

## Step 2: deploy your API

```bash
$ cd multi-model

$ cortex deploy
```

Wait for you API to be ready (you can track its progress with `cortex get --watch`).

## Step 3: make prediction requests

Run `cortex get text-analyzer` to get your API endpoint, and save it as a bash variable for convenience (yours will be different from mine):

```bash
$ api_endpoint=http://a36473270de8b46e79a769850dd3372d-c67035afa37ef878.elb.us-west-2.amazonaws.com/text-analyzer
```

Make a request to the sentiment analysis model:

```bash
$ curl ${api_endpoint}?model=sentiment -X POST -H "Content-Type: application/json" -d @sample-sentiment.json

{"label": "POSITIVE", "score": 0.9998506903648376}
```

Make a request to the text summarizer model:

```bash
$ curl ${api_endpoint}?model=summarizer -X POST -H "Content-Type: application/json" -d @sample-summarizer.json

Machine learning is the study of algorithms and statistical models that computer systems use to perform a specific task. It is seen as a subset of artificial intelligence. Machine learning algorithms are used in a wide variety of applications, such as email filtering and computer vision. In its application across business problems, machine learning is also referred to as predictive analytics.
```
