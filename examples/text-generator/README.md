# Deploy GPT-2 as a service

This example shows how to self-host OpenAI's GPT-2 model as a service on AWS.

## Define a deployment

Define a `deployment` and an `api` resource. A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes a model available as a web service that can serve real-time predictions. This configuration will download the model from the `cortex-examples` S3 bucket, preprocess the payload and postprocess the inference with functions defined in `encoder.py` and deploy each replica of the API on 1 GPU.

```yaml
- kind: deployment
  name: text

- kind: api
  name: generator
  model: s3://cortex-examples/text-generator/774
  request_handler: encoder.py
  compute:
    gpu: 1
```

## Add request handling

The model requires encoded data for inference, but the API should accept strings of natural language as input. It should also decode the model’s prediction before responding to the client. This can be implemented in a request handler file using the pre_inference and post_inference functions. See [encoder.py](encoder.py) for the complete code.

```python
# encoder = ...


def pre_inference(sample, metadata):
    context = encoder.encode(sample)
    return {"context": [context]}


def post_inference(prediction, metadata):
    return {encoder.decode(prediction["response"]["sample"])}
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from cortex.yaml and creates it on the cluster.

```bash
$ cortex deploy

Deployment started
```

Behind the scenes, Cortex containerizes the model, makes it servable using TensorFlow Serving, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

You can track the status of a deployment using cortex get:

```bash
$ cortex get --watch

api            replicas     last update
classifier     1/1          8s
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

```bash
$ curl http://***.amazonaws.com/text/generator \
    -X POST -H "Content-Type: application/json" \
    -d '{"samples": [{"machine learning"}]}'

Machine learning, with more than one thousand researchers around the world today, are looking to create computer-driven machine learning algorithms that can also be applied to human and social problems, such as education, health care, employment, medicine, politics, or the environment...
```

Any questions? [contact us](hello@cortex.dev) (we'll respond quickly).
