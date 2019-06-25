# Tutorial

## Prerequisites

1. A Kubernetes cluster running Cortex ([installation instructions](../cluster/install.md))
2. The Cortex CLI

## Deployment

### cortex.yaml

```text
$ mkdir iris && cd iris
$ touch cortex.yaml irises.json
```

Cortex requires a `cortex.yaml` file which defines a `deployment` resource. Other resources may be defined in arbitrarily named YAML files in the the directory which contains `cortex.yaml` or any subdirectories. We will define all of our resources in `cortex.yaml` in this example.

Add to `cortex.yaml`:

```yaml
- kind: deployment
  name: iris
```

### Define an API

An API makes your model available as a live web service that can serve real-time predictions. Cortex is able to read from any S3 bucket that your AWS credentials grant access to.

Add to `cortex.yaml`:

```yaml
# APIs

- kind: api
  name: iris-type
  external_model:
    path: s3://cortex-examples/iris-tensorflow.zip
    region: us-west-2
  compute:
    replicas: 3
```

### Deploy the API

```text
$ cortex deploy
```

You can get a summary of the status of resources using `cortex get`:

```text
$ cortex get --watch
```

### Test the iris classification service

Define a sample in `irises.json`:

```javascript
{
  "samples": [
    {
      "sepal_length": 5.2,
      "sepal_width": 3.6,
      "petal_length": 1.4,
      "petal_width": 0.3
    }
  ]
}
```

When the API is ready, request a prediction from the API:

```text
$ cortex predict iris-type irises.json
```

### Call the API from other clients (e.g. cURL)

Get the API's endpoint:

```text
$ cortex get iris-type
```

Use cURL to test the API:

```text
$ curl -k -X POST -H "Content-Type: application/json" \
       -d '{ "samples": [ { "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 } ] }' \
       <API endpoint>
```

### Cleanup

Delete the deployment:

```text
$ cortex delete iris
```

See [uninstall](../cluster/uninstall.md) if you'd like to uninstall Cortex.
