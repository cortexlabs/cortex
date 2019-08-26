# Deploy an iris classifier

This example shows how to deploy a classifier trained on the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris).

## Define a deployment

Define a `deployment` and an `api` resource in `cortex.yaml`. A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes a model available as a web service that can serve real-time predictions. This configuration will download the model from the `cortex-examples` S3 bucket.

```yaml
- kind: deployment
  name: iris

- kind: api
  name: classifier
  model: s3://cortex-examples/iris/tensorflow.zip
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

Deployment started
```

You can track the status of a deployment using `cortex get`:

```bash
$ cortex get --watch

api           replicas     last update
generator     1/1          8s
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

```bash
$ cortex get classifier
url: http://***.amazonaws.com/iris/classifier

$ curl http://***.amazonaws.com/iris/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{ "samples": [{"sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3}]}'
```
