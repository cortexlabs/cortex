# Deploy an iris classifier

This example demonstrates how to deploy a classifier trained on the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris).

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

```bash
$ cortex deploy
```

## Serve real-time predictions

```bash
$ curl http://***.amazonaws.com/iris/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{ "samples": [{"sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3}]}'
```
