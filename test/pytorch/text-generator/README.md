# Deploy machine learning models to production

_WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.24.*, run `git checkout -b 0.24` or switch to the `0.24` branch on GitHub)_

This example shows how to deploy a realtime text generation API using a GPT-2 model from Hugging Face's transformers library.

## Implement your Predictor

1. Create a Python file named `predictor.py`.
2. Define a Predictor class with a constructor that loads and initializes the model.
3. Add a predict function that will accept a payload and return the generated text.

```python
# predictor.py

import torch
from transformers import GPT2Tokenizer, GPT2LMHeadModel


class PythonPredictor:
    def __init__(self, config):
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")
        self.model = GPT2LMHeadModel.from_pretrained("gpt2").to(self.device)

    def predict(self, payload):
        input_length = len(payload["text"].split())
        tokens = self.tokenizer.encode(payload["text"], return_tensors="pt").to(self.device)
        prediction = self.model.generate(tokens, max_length=input_length + 20, do_sample=True)
        return self.tokenizer.decode(prediction[0])
```

## Specify Python dependencies

Create a `requirements.txt` file to specify the dependencies needed by `predictor.py`. Cortex will automatically install them into your runtime once you deploy:

```python
# requirements.txt

torch
transformers==3.0.*
```

## Deploy your model locally

You can create APIs from any Python runtime that has access to Docker (e.g. the Python shell or a Jupyter notebook):

```python
import cortex

cx_local = cortex.client("local")

api_spec = {
  "name": "text-generator",
  "kind": "RealtimeAPI",
  "predictor": {
    "type": "python",
    "path": "predictor.py"
  }
}

cx_local.create_api(api_spec, project_dir=".", wait=True)
```

## Consume your API

```python
import requests

endpoint = cx_local.get_api("text-generator")["endpoint"]
payload = {"text": "hello world"}
print(requests.post(endpoint, payload).text)
```

## Manage your APIs using the CLI

Monitor the status of your API using `cortex get`:

```bash
$ cortex get --watch

env     realtime api     status     last update   avg request   2XX
local   text-generator   updating   8s            -             -
```

Show additional information for your API (e.g. its endpoint) using `cortex get <api_name>`:

```bash
$ cortex get text-generator

status   last update   avg request   2XX
live     1m            -             -

endpoint: http://localhost:8889
```

You can also stream logs from your API:

```bash
$ cortex logs text-generator

...
```

## Deploy your model to AWS

Cortex can automatically provision infrastructure on your AWS account and deploy your models as production-ready web services:

```bash
$ cortex cluster up
```

This creates a Cortex cluster in your AWS account, which will take approximately 15 minutes. After your cluster is created, you can deploy to your cluster by using the same code and configuration as before:

```python
import cortex

cx_aws = cortex.client("aws")

api_spec = {
  "name": "text-generator",
  "kind": "RealtimeAPI",
  "predictor": {
    "type": "python",
    "path": "predictor.py"
  }
}

cx_aws.create_api(api_spec, project_dir=".")
```

Monitor the status of your APIs using `cortex get` using your CLI:

```bash
$ cortex get --watch

env     realtime api     status   up-to-date   requested   last update   avg request   2XX
aws     text-generator   live     1            1           1m            -             -
local   text-generator   live     1            1           17m           3.1285 s      1
```

The output above indicates that one replica of your API was requested and is available to serve predictions. Cortex will automatically launch more replicas if the load increases and will spin down replicas if there is unused capacity.

Show additional information for your API (e.g. its endpoint) using `cortex get <api_name>`:

```bash
$ cortex get text-generator --env aws

status   up-to-date   requested   last update   avg request   2XX
live     1            1           1m            -             -

endpoint: https://***.execute-api.us-west-2.amazonaws.com/text-generator
```

## Run on GPUs

If your cortex cluster is using GPU instances (configured during cluster creation) or if you are running locally with an nvidia GPU, you can run your text generator API on GPUs. Add the `compute` field to your API configuration and re-deploy:

```python
api_spec = {
  "name": "text-generator",
  "kind": "RealtimeAPI",
  "predictor": {
    "type": "python",
    "path": "predictor.py"
  },
  "compute": {
    "gpu": 1
  }
}

cx_aws.create_api(api_spec, project_dir=".")
```

As your new API is initializing, the old API will continue to respond to prediction requests. Once the API's status becomes "live" (with one up-to-date replica), traffic will be routed to the updated version. You can track the status of your API using `cortex get`:

```bash
$ cortex get --env aws --watch

realtime api     status     up-to-date   stale   requested   last update   avg request   2XX
text-generator   updating   0            1       1           29s           -             -
```

## Cleanup

Deleting APIs will free up cluster resources and allow Cortex to scale down to the minimum number of instances you specified during cluster creation:

```python
cx_local.delete_api("text-generator")

cx_aws.delete_api("text-generator")
```
