# Deploy an iris classifier

This example shows how to deploy a classifier trained on the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris).

## Define a deployment

Define a `deployment` and an `api` resource in `cortex.yaml`. A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes a model available as a web service that can serve real-time predictions. This configuration will download the model from the `cortex-examples` S3 bucket.

```yaml
- kind: deployment
  name: iris

- kind: api
  name: classifier
  model: s3://cortex-examples/iris/tensorflow/1560263532
  request_handler: handlers/tensorflow.py
```

## Add request handling

The API should convert the model’s prediction to a human readable label before responding to the client. This can be implemented in a request handler file:

```python
labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def post_inference(prediction, metadata):
    label_index = int(prediction["response"]["class_ids"][0])
    return labels[label_index]
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

api          available   up-to-date   requested   last update
classifier   1           1            1           8s
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

```bash
$ cortex get classifier

url: http://***.amazonaws.com/iris/classifier

$ curl http://***.amazonaws.com/iris/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3}'

iris-setosa
```

Any questions? [contact us](hello@cortex.dev) (we'll respond quickly).
