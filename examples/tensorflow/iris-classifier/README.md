# Deploy an iris classifier

This example shows how to deploy a classifier trained on the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris).

## Define a deployment

A `deployment` specifies a set of resources that are deployed together. An `api` makes our exported model available as a web service that can serve real-time predictions. This configuration will deploy our model from the `cortex-examples` S3 bucket:

```yaml
# cortex.yaml

- kind: deployment
  name: iris

- kind: api
  name: classifier
  tensorflow:
    model: s3://cortex-examples/tensorflow/iris-classifier/nn
    request_handler: handler.py
  tracker:
    model_type: classification
```

<!-- CORTEX_VERSION_MINOR -->
You can run the code that generated the exported model used in this example [here](https://colab.research.google.com/github/cortexlabs/cortex/blob/master/examples/tensorflow/iris-classifier/tensorflow.ipynb).

## Add request handling

The API should convert the model’s prediction to a human readable label before responding. This can be implemented in a request handler file:

```python
# handler.py

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

def post_inference(prediction, metadata):
    label_index = int(prediction["class_ids"][0])
    return labels[label_index]
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes our implementation, makes it servable using Flask, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

We can track the status of a deployment using `cortex get`:

```bash
$ cortex get classifier --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           8s            -
```

The output above indicates that one replica of the API was requested and is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

We can use `curl` to test our prediction service:

```bash
$ cortex get classifier

url: http://***.amazonaws.com/iris/classifier

$ curl http://***.amazonaws.com/iris/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3}'

"iris-setosa"
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
