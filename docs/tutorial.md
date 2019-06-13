# Tutorial

## Prerequisites

1. An AWS account
1. A Kubernetes cluster running the Cortex operator ([installation instructions](operator/install.md))
1. The Cortex CLI

## Build a machine learning application

Let's build and deploy a classifier using the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris)! Below are a few samples of iris data:

|sepal_length|sepal_width|petal_length|petal_width|class|
|:---:|:---:|:---:|:---:|:---|
|5.1|3.5|1.4|0.2|Iris-setosa|
|7.0|3.2|4.7|1.4|Iris-versicolor|
|6.3|3.3|6.0|2.5|Iris-virginica|

Our goal is to build a web API that returns the type of iris given its measurements.

#### Initialize the application

```bash
mkdir iris && cd iris
touch app.yaml dnn.py irises.json
```

Cortex requires an `app.yaml` file which defines a single `app` resource. Other resources may be defined in arbitrarily named YAML files in the the directory which contains `app.yaml` or any subdirectories. For this example, we will define all of our resources in `app.yaml`.

Add to `app.yaml`:

```yaml
- kind: app
  name: iris
```

#### Configure data ingestion

Add to `app.yaml`:

```yaml
# Environments

- kind: environment
  name: dev
  data:
    type: csv
    path: s3a://cortex-examples/iris.csv
    schema: [@sepal_length, @sepal_width, @petal_length, @petal_width, @class]
```

Cortex will be able to read from any S3 bucket that your AWS credentials grant access to.

#### Define the model

This configuration will generate a training dataset with the specified columns and train our classifier using the generated dataset. Here we're using TensorFlow's [DNNClassifier](https://www.tensorflow.org/api_docs/python/tf/estimator/DNNClassifier) but Cortex supports any TensorFlow code that adheres to the [tf.estimator API](https://www.tensorflow.org/guide/estimators).

Add to `app.yaml`:

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

#### Define web APIs

This will make the model available as a live web service that can serve real-time predictions.

Add to `app.yaml`:

```yaml
# APIs

- kind: api
  name: iris-type
  model: @dnn
  compute:
    replicas: 1
```

## Deploy the application

```
$ cortex deploy

Deployment started
```

You can get a summary of the status of resources using `cortex status`:

```
$ cortex status --watch

Python Packages:       none
Raw Columns:           5 ready
Aggregates:            9 ready
Transformed Columns:   5 ready
Training Datasets:     1 ready
Models:                1 training
APIs:                  1 pending
```

#### Test the iris classification service

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

```
$ cortex predict iris-type irises.json

iris-type was last updated on Tuesday, February 14, 2019 at 2:57pm PST

Predicted class:
Iris-setosa
```

#### Call the API from other clients (e.g. cURL)

Get the API's endpoint:

```
$ cortex get api iris-type

# https://abc.amazonaws.com/iris/iris-type
```

Use cURL to test the API:

```
$ curl -k \
     -X POST \
     -H "Content-Type: application/json" \
     -d '{ "samples": [ { "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 } ] }' \
     <API endpoint>

{"classification_predictions":[{"class_ids":["0"],"classes":["MA=="],"logits":[1.501487135887146,-0.6141998171806335,-1.4335800409317017],"predicted_class":0,"predicted_class_reversed":"Iris-setosa","probabilities":[0.8520227670669556,0.10271172970533371,0.04526554048061371]}],"resource_id":"18ef9f6fb4a1a8b2a3d3e8068f179f89f65d1ae3d8ac9d96b782b1cec3b39d2"}
```

## Cleanup

Delete the iris application:

```
$ cortex delete iris

Deployment deleted
```

See [uninstall](operator/uninstall.md) if you'd like to uninstall Cortex.
