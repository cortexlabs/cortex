# Self-host HuggingFace's GPT-2 as a service

This example shows how to deploy [HuggingFace's DistilGPT2](https://github.com/huggingface/transformers/tree/master/examples/distillation) model as a service on AWS. DistilGPT2 is a compressed version of OpenAI's GPT-2.

## Predictor

We implement Cortex's Predictor interface to load the model and make predictions. Cortex will use this implementation to serve the model as an autoscaling API.

### Initialization

We can place our code to download and initialize the model in the body of the implementation, and we can load it onto a GPU in the `init()` function:

```python
# predictor.py

# download the pretrained DistilGPT2 model and set it to evaluation
model = GPT2LMHeadModel.from_pretrained("distilgpt2")
model.eval()

# download the tokenizer
tokenizer = GPT2Tokenizer.from_pretrained("distilgpt2")

def init(metadata):
    # load the model onto the device specified in the metadata field of our api configuration
    model.to(metadata["device"])
```

### Predict

The `predict()` function will be triggered once per request. We tokenize the input, run it through the model, decode the output, and respond with the generated text.

```python
# predictor.py

def predict(sample, metadata):
    indexed_tokens = tokenizer.encode(sample["text"])
    output = sample_sequence(model, metadata['num_words'], indexed_tokens, device=metadata['device'])
    return tokenizer.decode(
        output[0, 0:].tolist(), clean_up_tokenization_spaces=True, skip_special_tokens=True
    )
```

See [predictor.py](./predictor.py) for the complete code.

## Define a deployment

A `deployment` specifies a set of resources that are deployed together. An `api` makes our implementation available as a web service that can serve real-time predictions. This configuration will deploy the implementation specified in `predictor.py`:

```yaml
# cortex.yaml

- kind: deployment
  name: text

- kind: api
  name: generator
  predictor:
    path: predictor.py
    metadata:
      num_words: 20  # generate 20 words per request
      device: cuda  # run on GPU
  compute:
    gpu: 1
    cpu: 1
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster.

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes our implementation, makes it servable using Flask, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

We can track the status of a deployment using `cortex get`:

```bash
$ cortex get generator --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           9m            -
```

The output above indicates that one replica of the API was requested and is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

We can use `curl` to test our prediction service:

```bash
$ cortex get generator

url: http://***.amazonaws.com/text/generator

$ curl http://***.amazonaws.com/text/generator \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "machine learning"}'
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
