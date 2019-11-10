# Self-host OpenAI's GPT-2 as a service

This example shows how to deploy OpenAI's GPT-2 model as a service on AWS.

## Define a deployment

A `deployment` specifies a set of resources that are deployed together. An `api` makes our exported model available as a web service that can serve real-time predictions. This configuration will download the 124M GPT-2 model from the `cortex-examples` S3 bucket, preprocess the payload and postprocess the inference with functions defined in `handler.py`, and deploy each replica of the API on 1 GPU:

```yaml
# cortex.yaml

- kind: deployment
  name: text

- kind: api
  name: generator
  tensorflow:
    model: s3://cortex-examples/text-generator/gpt-2/124M
    request_handler: handler.py
  compute:
    cpu: 1
    gpu: 1
```

<!-- CORTEX_VERSION_MINOR -->
You can run the code that generated the exported GPT-2 model [here](https://colab.research.google.com/github/cortexlabs/cortex/blob/master/examples/tensorflow/text-generator/gpt-2.ipynb).

## Add request handling

The model requires encoded data for inference, but the API should accept strings of natural language as input. It should also decode the inference output as human-readable text.

```python
# handler.py

from encoder import get_encoder
encoder = get_encoder()

def pre_inference(sample, signature, metadata):
    context = encoder.encode(sample["text"])
    return {"context": [context]}

def post_inference(prediction, signature, metadata):
    response = prediction["sample"]
    return encoder.decode(response)
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
live     1            1           1           8s            -
```

The output above indicates that one replica of the API was requested and is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

```bash
$ cortex get generator

We can use `curl` to test our prediction service:

endpoint: http://***.amazonaws.com/text/generator

$ curl http://***.amazonaws.com/text/generator \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "machine learning"}'

Machine learning, with more than one thousand researchers around the world today, are looking to create computer-driven machine learning algorithms that can also be applied to human and social problems, such as education, health care, employment, medicine, politics, or the environment...
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
