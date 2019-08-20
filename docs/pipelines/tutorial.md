# Pipeline Tutorial

## Prerequisites

1. A Cortex cluster ([installation instructions](../cluster/install.md))
2. The Cortex CLI ([installation instructions](../cluster/install.md))

## Deployment

Let's build and deploy a classifier using the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris)! Below are a few samples of iris data:

|sepal_length|sepal_width|petal_length|petal_width|class|
|:---:|:---:|:---:|:---:|:---|
|5.1|3.5|1.4|0.2|Iris-setosa|
|7.0|3.2|4.7|1.4|Iris-versicolor|
|6.3|3.3|6.0|2.5|Iris-virginica|

Our goal is to build a web API that returns the type of iris given its measurements.

### cortex.yaml

```text
mkdir iris && cd iris && touch cortex.yaml
```

Cortex requires an `cortex.yaml` file which defines a `deployment` resource. Other resources may be defined in arbitrarily named YAML files in the the directory which contains `cortex.yaml` or any subdirectories. For this example, we will define all of our resources in `cortex.yaml`.

Add to `cortex.yaml`:

```yaml
- kind: deployment
  name: iris
```

### Configure data ingestion

Add to `cortex.yaml`:

```yaml
# Environments

- kind: environment
  name: dev
  data:
    type: csv
    path: s3a://cortex-examples/iris.csv
    schema: [@sepal_length, @sepal_width, @petal_length, @petal_width, @class]
```

Cortex is able to read from any S3 bucket that you have access to.

### Define the model

This configuration will generate a training dataset with the specified columns and train our classifier using the generated dataset. Here we're using a built-in estimator (which uses TensorFlow's [DNNClassifier](https://www.tensorflow.org/api_docs/python/tf/estimator/DNNClassifier)) but Cortex supports any TensorFlow code that adheres to the [tf.estimator API](https://www.tensorflow.org/guide/estimators).

Add to `cortex.yaml`:

```yaml
# Models

- kind: model
  name: dnn
  estimator: cortex.dnn_classifier
  target_column: @class
  input:
    numeric_columns: [@sepal_length, @sepal_width, @petal_length, @petal_width]
    target_vocab: ['Iris-setosa', 'Iris-versicolor', 'Iris-virginica']
  hparams:
    hidden_units: [4, 2]
  training:
    batch_size: 10
    num_steps: 1000
```

### Define web APIs

This will make the model available as a live web service that can serve real-time predictions.

Add to `cortex.yaml`:

```yaml
# APIs

- kind: api
  name: iris-type
  model: @dnn
```

## Deploy the pipeline

```text
$ cortex deploy

Deployment started
```

You can get a summary of the status of resources using `cortex get`:

```text
$ cortex get --watch
```

### Test the API

Get the API's endpoint:

```text
$ cortex get api iris-type
```

Use cURL to test the API:

```text
$ curl -k \
     -X POST \
     -H "Content-Type: application/json" \
     -d '{ "samples": [ { "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 } ] }' \
     <API endpoint>
```

## Cleanup

Delete the iris application:

```text
$ cortex delete iris

Deployment deleted
```
