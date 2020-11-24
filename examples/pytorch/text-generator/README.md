# Deploy machine learning models to production

_WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.22.*, run `git checkout -b 0.22` or switch to the `0.22` branch on GitHub)_

This example shows how to deploy a realtime text generation API using a GPT-2 model from Hugging Face's transformers library.

## Implement your Predictor

Define a Predictor with an `__init__` function that loads and initializes the model, and a `predict` function that accepts a payload and returns text.

```python
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

Create a list of dependencies for your Predictor. Cortex will automatically install them into your runtime once you deploy.

```python
pip_dependencies = ["torch", "transformers==3.0.*"]
```

## Configure your API

A `RealtimeAPI` provides a runtime for inference and makes your Predictor implementation available as a web service that can serve realtime predictions.

```python
api_spec = {
  'name': 'text-generator',
  'kind': 'RealtimeAPI'
}
```

## Deploy your API locally

`deploy` uses the Predictor implementation and API configuration to create a web API:

```python
import cortex

cx = cortex.client("local")
cx.deploy(api_spec, predictor=PythonPredictor)
```

## Make a request to your local API

```python
import requests

endpoint = cx.get("text-generator")["endpoint"]
payload = {"text": "hello world"}
print(requests.post(endpoint, payload))
```

## Manage your APIs

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
...
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

This creates a Cortex cluster in your AWS account, which will take approximately 15 minutes. After your cluster is created, you can deploy your model to your cluster by using the same code and configuration as before:

```python
cx = cortex.client("aws")
cx.deploy(api_spec, predictor=PythonPredictor)
```

Monitor the status of your APIs using `cortex get`:

```bash
$ cortex get --watch

env     realtime api     status   up-to-date   requested   last update   avg request   2XX
aws     text-generator   live     1            1           1m            -             -
local   text-generator   live     1            1           17m           3.1285 s      1
```

The output above indicates that one replica of your API was requested and is available to serve predictions. Cortex will automatically launch more replicas if the load increases and will spin down replicas if there is unused capacity.

## Make a request to your AWS API

```python
import requests

endpoint = cx.get("text-generator")["endpoint"]
payload = {"text": "hello world"}
print(requests.post(endpoint, payload))
```

## Perform a rolling update

When you make a change to your Predictor or your API configuration, you can update your api by re-running `cx.deploy()`.

Let's modify the Predictor to set the length of the generated text based on a query parameter:

```python
import torch
from transformers import GPT2Tokenizer, GPT2LMHeadModel

class PythonPredictor:
    def __init__(self, config):
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")
        self.model = GPT2LMHeadModel.from_pretrained("gpt2").to(self.device)

    def predict(self, payload, query_params):  # this line is updated
        input_length = len(payload["text"].split())
        output_length = int(query_params.get("length", 20))  # this line is added
        tokens = self.tokenizer.encode(payload["text"], return_tensors="pt").to(self.device)
        prediction = self.model.generate(tokens, max_length=input_length + output_length, do_sample=True)  # this line is updated
        return self.tokenizer.decode(prediction[0])
```

Perform a rolling update of your API.

```python
cx = cortex.client("aws")
cx.deploy(api_spec, predictor=PythonPredictor)
```

You can track the status of your API using `cortex get`:

```bash
$ cortex get --env aws --watch

realtime api     status     up-to-date   stale   requested   last update   avg request   2XX
text-generator   updating   0            1       1           29s           -             -
```

As your new implementation is initializing, the old implementation will continue to be used to respond to prediction requests. Eventually the API's status will become "live" (with one up-to-date replica), and traffic will be routed to the updated version.

```python
import requests

endpoint = cx.get("text-generator")["endpoint"]
payload = {"text": "hello world"}
print(requests.post(endpoint, payload))
```

## Run on GPUs

If your cortex cluster is using GPU instances (configured during cluster creation), you can run your text generator API on GPUs. Add the `compute` field to your API configuration:

```python
api_spec = {
  'name': 'text-generator',
  'kind': 'RealtimeAPI',
  'compute': {
    'gpu': 1
  }
}
```

Run `deploy` to update your API with this configuration:

```python
cx.deploy(api_spec, predictor=PythonPredictor)
```

You can use `cortex get` to check the status of your API, and once it's live, prediction requests should be faster.

## Disable rolling updates

You can disable rolling updates to prevent spinning up extra instances in development environments by setting `max_surge` to 0 in the `update_strategy` configuration.

```python

config = {
  'name': 'text-generator',
  'kind': 'RealtimeAPI',
  'compute': {
    'gpu': 1
  },
  'update_strategy': {
    'max_surge': 0
  }
}
```

## Cleanup

Run `delete` to delete each API:

```python
cortex.delete('text-generator', 'local')

cortex.delete('text-generator', 'aws')
```

Deleting APIs frees compute resources and allow Cortex to scale down to the minimum number of instances you specified during cluster creation. `cortex.delete()` will not spin down your cluster.
