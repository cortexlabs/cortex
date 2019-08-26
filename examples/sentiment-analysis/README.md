# Deploy a BERT sentiment analysis API

This example shows how to deploy a sentiment analysis classifier trained using [BERT](https://github.com/google-research/bert).

## Define a deployment

Define a `deployment` and an `api` resource in `cortex.yaml`. A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes a model available as a web service that can serve real-time predictions. This configuration will download the model from the `cortex-examples` S3 bucket.

```yaml
- kind: deployment
  name: sentiment

- kind: api
  name: classifier
  model: s3://cortex-examples/sentiment/1565392692
  request_handler: sentiment.py
```

## Add request handling

The API should convert the model’s prediction to a human readable label before responding to the client. This can be implemented in a request handler file:

```python
import tensorflow as tf
import tensorflow_hub as hub
from bert import tokenization, run_classifier

labels = ["negative", "positive"]

with tf.Graph().as_default():
    bert_module = hub.Module("https://tfhub.dev/google/bert_uncased_L-12_H-768_A-12/1")
    info = bert_module(signature="tokenization_info", as_dict=True)
    with tf.Session() as sess:
        vocab_file, do_lower_case = sess.run([info["vocab_file"], info["do_lower_case"]])
tokenizer = tokenization.FullTokenizer(vocab_file=vocab_file, do_lower_case=do_lower_case)


def pre_inference(sample, metadata):
    input_example = run_classifier.InputExample(guid="", text_a=sample["input"], label=0)
    input_feature = run_classifier.convert_single_example(0, input_example, [0, 1], 128, tokenizer)
    return {"input_ids": [input_feature.input_ids]}


def post_inference(prediction, metadata):
    return labels[prediction["response"]["labels"][0]]
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

Deployment started
```

Behind the scenes, Cortex containerizes the model, makes it servable using TensorFlow Serving, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

You can track the status of a deployment using `cortex get`:

```bash
$ cortex get --watch

api          replicas     last update
analysis     1/1          8s
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

```bash
$ cortex get classifier

url: http://***.amazonaws.com/sentiment/analysis

$ curl http://***.amazonaws.com/sentiment/analysis \
    -X POST -H "Content-Type: application/json" \
    -d '{ "samples": [{"input": "The movie was great!"}]}'

positive
```

Any questions? [contact us](hello@cortex.dev) (we'll respond quickly).
