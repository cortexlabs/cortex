# Self-host HuggingFace's GPT-2 as a service

This example shows how to deploy HuggingFace's DistilGPT2 (Based on OpenAI's GPT-2) model as a service on AWS.


## Inference

We implement Cortex's python inference interface that describes how to load the model and make predictions using the model. Cortex will use this implementation to serve your model as an API of autoscaling replicas. We specify a `requirements.txt` to install dependencies necessary to implement the Cortex inference interface.

### Initialization

Cortex executes the python implementation once per replica startup. We can place our initializations in the body of the implementation. Let us download the pretrained DistilGPT2 model and set it to evaluation:
```
model = GPT2LMHeadModel.from_pretrained("distilgpt2")
model.eval()
tokenizer = GPT2Tokenizer.from_pretrained("distilgpt2")
```

### Predict
The `predict` function will be triggered once per request to run the text generation model on a prompt provided in the request. We tokenize the prompt and and generate the number of words specified in the deployment definition `cortex.yaml`. We decode the output of the model and respond with readable generated text.

```python
def sample_sequence(model, lenght, context, **kwargs):
    #...

def predict(sample, metadata):
    indexed_tokens = tokenizer.encode(sample["text"])
    output = sample_sequence(model, metadata['num_words'], indexed_tokens)
    return tokenizer.decode(
        output[0, 0:].tolist(), clean_up_tokenization_spaces=True, skip_special_tokens=True
    )
```

See `inference.py` for more details.

## Define a deployment

A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes the Cortex python implementation available as a web service that can serve real-time predictions.  This configuration will deploy the implentation specified in `inference.py` and generates 20 words per request.

```yaml
- kind: deployment
  name: gpt2

- kind: api
  name: text-gen
  python:
    inference: inference.py
  metadata:
    num_words: 20
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster.

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes the model, makes it servable using TensorFlow Serving, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

You can track the status of a deployment using `cortex get`:

```bash
$ cortex get text-gen --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           9m            -
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

```bash
$ cortex get text-gen

url: http://***.amazonaws.com/text/generator

$ curl http://***.amazonaws.com/text/generator \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "machine learning"}'

machine learning and simulated engine testing platform. The model can be applied in this article. We are implementing a workflow...
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
