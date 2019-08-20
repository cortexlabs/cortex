# Tutorial

## Prerequisites

1. A Cortex cluster ([installation instructions](../cluster/install.md))
2. The Cortex CLI ([installation instructions](../cluster/install.md))

## Deployment

Let's deploy a classifier built using the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris)!

### cortex.yaml

```text
$ mkdir iris && cd iris && touch cortex.yaml
```

Cortex requires a `cortex.yaml` file which defines a `deployment` resource. An `api` resource makes the model available as a live web service that can serve real-time predictions.

```yaml
# cortex.yaml

- kind: deployment
  name: iris

- kind: api
  name: classifier
  model: s3://cortex-examples/iris/tensorflow.zip
```

Cortex is able to read from any S3 bucket that you have access to.

### Deploy the API

```bash
$ cortex deploy

# Get a summary of resource statuses
$ cortex get --watch
```

### Test the API

```bash
# Get the endpoint
$ cortex get classifier

# Test the API
$ curl -k -X POST -H "Content-Type: application/json" \
       -d '{ "samples": [ { "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 } ] }' \
       <API endpoint>
```

### Cleanup

```bash
# Delete the deployment
cortex delete iris
```
