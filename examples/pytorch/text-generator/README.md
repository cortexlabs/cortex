# Deploy models as Realtime APIs

_WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.19.*, run `git checkout -b 0.19` or switch to the `0.19` branch on GitHub)_

This example shows how to deploy a realtime text generation API using a GPT-2 model from Hugging Face's transformers library.

<br>

## Implement your predictor

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
        print(f"using device: {self.device}")
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")
        self.model = GPT2LMHeadModel.from_pretrained("gpt2").to(self.device)

    def predict(self, payload):
        input_length = len(payload["text"].split())
        tokens = self.tokenizer.encode(payload["text"], return_tensors="pt").to(self.device)
        prediction = self.model.generate(tokens, max_length=input_length + 20, do_sample=True)
        return self.tokenizer.decode(prediction[0])
```

Here are the complete [Predictor docs](../../../docs/deployments/realtime-api/predictors.md).

<br>

## Specify your Python dependencies

Create a `requirements.txt` file to specify the dependencies needed by `predictor.py`. Cortex will automatically install them into your runtime once you deploy:

```python
# requirements.txt

torch
transformers==3.0.*
```

<br>

## Configure your API

Create a `cortex.yaml` file and add the configuration below. A `RealtimeAPI` provides a runtime for inference and makes your `predictor.py` implementation available as a web service that can serve real-time predictions:

```yaml
# cortex.yaml

- name: text-generator
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
```

Here are the complete [API configuration docs](../../../docs/deployments/realtime-api/api-configuration.md).

<br>

## Deploy your model locally

`cortex deploy` takes your Predictor implementation along with the configuration from `cortex.yaml` and creates a web API:

```bash
$ cortex deploy

creating text-generator (RealtimeAPI)
```

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

endpoint: http://localhost:8888
...
```

You can also stream logs from your API:

```bash
$ cortex logs text-generator

...
```

Once your API is live, use `curl` to test your API (it will take a few seconds to generate the text):

```bash
$ curl http://localhost:8888 \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "machine learning is"}'

"machine learning is ..."
```

<br>

## Deploy your model to AWS

Cortex can automatically provision infrastructure on your AWS account and deploy your models as production-ready web services:

```bash
$ cortex cluster up
```

This creates a Cortex cluster in your AWS account, which will take approximately 15 minutes.

After your cluster is created, you can deploy your model to your cluster by using the same code and configuration as before:

```bash
$ cortex deploy --env aws

creating text-generator (RealtimeAPI)
```

_Note that the `--env` flag specifies the name of the CLI environment to use. [CLI environments](../../../docs/miscellaneous/environments.md) contain the information necessary to connect to your cluster. The default environment is `local`, and when the cluster was created, a new environment named `aws` was created to point to the cluster. You can change the default environment with `cortex env default <env_name`)._

Monitor the status of your APIs using `cortex get`:

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
live     1            1           17m           -             -

metrics dashboard: https://us-west-2.console.aws.amazon.com/cloudwatch/home#dashboards:name=cortex
endpoint: https://***.execute-api.us-west-2.amazonaws.com/text-generator
...
```

Use your new endpoint to make requests to your API on AWS:

```bash
$ curl https://***.execute-api.us-west-2.amazonaws.com/text-generator \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "machine learning is"}'

"machine learning is ..."
```

<br>

## Perform a rolling update

When you make a change to your `predictor.py` or your `cortex.yaml`, you can update your api by re-running `cortex deploy`.

Let's modify `predictor.py` to set the length of the generated text based on a query parameter:

```python
# predictor.py

import torch
from transformers import GPT2Tokenizer, GPT2LMHeadModel


class PythonPredictor:
    def __init__(self, config):
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"using device: {self.device}")
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")
        self.model = GPT2LMHeadModel.from_pretrained("gpt2").to(self.device)

    def predict(self, payload, query_params):  # this line is updated
        input_length = len(payload["text"].split())
        output_length = int(query_params.get("length", 20))  # this line is added
        tokens = self.tokenizer.encode(payload["text"], return_tensors="pt").to(self.device)
        prediction = self.model.generate(tokens, max_length=input_length + output_length, do_sample=True)  # this line is updated
        return self.tokenizer.decode(prediction[0])
```

Run `cortex deploy` to perform a rolling update of your API:

```bash
$ cortex deploy --env aws

updating text-generator (RealtimeAPI)
```

You can track the status of your API using `cortex get`:

```bash
$ cortex get --env aws --watch

realtime api     status     up-to-date   stale   requested   last update   avg request   2XX
text-generator   updating   0            1       1           29s           -             -
```

As your new implementation is initializing, the old implementation will continue to be used to respond to prediction requests. Eventually the API's status will become "live" (with one up-to-date replica), and traffic will be routed to the updated version.

Try your new code:

```bash
$ curl https://***.execute-api.us-west-2.amazonaws.com/text-generator?length=30 \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "machine learning is"}'

"machine learning is ..."
```

<br>

## Run on GPUs

If your cortex cluster is using GPU instances (configured during cluster creation), you can run your text generator API on GPUs. Add the `compute` field to your API configuration:

```yaml
# cortex.yaml

- name: text-generator
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
  compute:
    gpu: 1
```

Run `cortex deploy` to update your API with this configuration:

```bash
$ cortex deploy --env aws

updating text-generator (RealtimeAPI)
```

You can use `cortex get` to check the status of your API, and once it's live, prediction requests should be faster.

### A note about rolling updates in dev environments

In development environments, you may wish to disable rolling updates since rolling updates require additional cluster resources. For example, a rolling update of a GPU-based API will require at least two GPUs, which can require a new instance to spin up if your cluster only has one instance. To disable rolling updates, set `max_surge` to 0 in the `update_strategy` configuration:

```yaml
# cortex.yaml

- name: text-generator
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
  compute:
    gpu: 1
  update_strategy:
    max_surge: 0
```

<br>

## Cleanup

Run `cortex delete` to delete each API:

```bash
$ cortex delete text-generator --env local

deleting text-generator

$ cortex delete text-generator --env aws

deleting text-generator
```

Running `cortex delete` will free up cluster resources and allow Cortex to scale down to the minimum number of instances you specified during cluster creation. It will not spin down your cluster.

## Next steps

<!-- CORTEX_VERSION_MINOR -->
* Deploy another one of our [examples](https://github.com/cortexlabs/cortex/tree/master/examples).
* See our [exporting guide](../../../docs/guides/exporting.md) for how to export your model to use in an API.
* Try the [batch API tutorial](../../batch/image-classifier/README.md) to learn how to deploy batch APIs in Cortex.
* See our [traffic splitter example](../../traffic-splitter/README.md) for how to deploy multiple APIs and set up a traffic splitter.
* See [uninstall](../../../docs/cluster-management/uninstall.md) if you'd like to spin down your cluster.
