# Quick Start

## Prerequisites

1. AWS account
2. Kubernetes cluster running Cortex ([installation instructions](install.md))
3. Cortex CLI

## TL;DR

You can download the pre-built iris application by cloning our repository:

<!-- CORTEX_VERSION_MINOR -->

```bash
git clone -b master https://github.com/cortexlabs/cortex.git
cd cortex/examples/iris
```

Jump to [Deploy the application](#deploy-the-application).

## Build a machine learning application

Let's build and deploy a classifier using the famous [Iris Data Set](https://archive.ics.uci.edu/ml/datasets/iris)!

#### Initialize the application

```bash
cortex init iris
cd iris
```

The following directories and files have been created:

```text
./iris/
├── app.yaml
├── samples.json
├── implementations
│   ├── aggregators
│   │   └── aggregator.py
│   ├── models
│   │   └── model.py
│   └── transformers
│       └── transformer.py
└── resources
    ├── aggregators.yaml
    ├── aggregates.yaml
    ├── apis.yaml
    ├── constants.yaml
    ├── environments.yaml
    ├── models.yaml
    ├── raw_features.yaml
    ├── transformed_features.yaml
    └── transformers.yaml
```

#### Name the application

```yaml
# app.yaml

- kind: app
  name: iris
```


#### Configure data ingestion

```yaml
# resources/environments.yaml

- kind: environment
  name: dev
  data:
    type: csv
    path: s3a://cortex-examples/iris.csv
    schema:
      - sepal_length
      - sepal_width
      - petal_length
      - petal_width
      - class
```

Cortex will be able to read from any S3 bucket that your AWS credentials grant access to.

#### Define raw features

```yaml
# resources/raw_features.yaml

- kind: raw_feature
  name: sepal_length
  type: FLOAT_FEATURE
  min: 0
  max: 10

- kind: raw_feature
  name: sepal_width
  type: FLOAT_FEATURE
  min: 0
  max: 10

- kind: raw_feature
  name: petal_length
  type: FLOAT_FEATURE
  min: 0
  max: 10

- kind: raw_feature
  name: petal_width
  type: FLOAT_FEATURE
  min: 0
  max: 10

- kind: raw_feature
  name: class
  type: STRING_FEATURE
  values: ['Iris-setosa', 'Iris-versicolor', 'Iris-virginica']
```

The Iris Data Set consists of four attributes and a label. We ensure that the data matches the types we expect, the numerical data is within a reasonable range, and the class labels are within the set of expected labels.

#### Define aggregates

Aggregates are computations that require processing the full column. We want to normalize the numeric features, so we need mean and standard deviation values for each numeric column. We also need a mapping of strings to integers for the label column. Cortex has `mean`, `stddev`, and `index_string` aggregators out of the box.

```yaml
# resources/aggregates.yaml

- kind: aggregate
  name: sepal_length_mean
  aggregator: cortex.mean
  inputs:
    features:
      col: sepal_length

- kind: aggregate
  name: sepal_length_stddev
  aggregator: cortex.stddev
  inputs:
    features:
      col: sepal_length

- kind: aggregate
  name: sepal_width_mean
  aggregator: cortex.mean
  inputs:
    features:
      col: sepal_width

- kind: aggregate
  name: sepal_width_stddev
  aggregator: cortex.stddev
  inputs:
    features:
      col: sepal_width

- kind: aggregate
  name: petal_length_mean
  aggregator: cortex.mean
  inputs:
    features:
      col: petal_length

- kind: aggregate
  name: petal_length_stddev
  aggregator: cortex.stddev
  inputs:
    features:
      col: petal_length

- kind: aggregate
  name: petal_width_mean
  aggregator: cortex.mean
  inputs:
    features:
      col: petal_width

- kind: aggregate
  name: petal_width_stddev
  aggregator: cortex.stddev
  inputs:
    features:
      col: petal_width

- kind: aggregate
  name: class_index
  aggregator: cortex.index_string
  inputs:
    features:
      col: class
```

#### Define transformed features

Transformers convert the raw features into the appropriate inputs for a TensorFlow model. Here we use the built-in `normalize` and `index_string` transformers using the aggregates we computed earlier.

```yaml
# resources/transformed_features.yaml

- kind: transformed_feature
  name: sepal_length_normalized
  transformer: cortex.normalize
  inputs:
    features:
      num: sepal_length
    args:
      mean: sepal_length_mean
      stddev: sepal_length_stddev

- kind: transformed_feature
  name: sepal_width_normalized
  transformer: cortex.normalize
  inputs:
    features:
      num: sepal_width
    args:
      mean: sepal_width_mean
      stddev: sepal_width_stddev

- kind: transformed_feature
  name: petal_length_normalized
  transformer: cortex.normalize
  inputs:
    features:
      num: petal_length
    args:
      mean: petal_length_mean
      stddev: petal_length_stddev

- kind: transformed_feature
  name: petal_width_normalized
  transformer: cortex.normalize
  inputs:
    features:
      num: petal_width
    args:
      mean: petal_width_mean
      stddev: petal_width_stddev

- kind: transformed_feature
  name: class_indexed
  transformer: cortex.index_string
  inputs:
    features:
      text: class
    args:
      index: class_index
```

You can simplify this YAML using [templates](applications/advanced/templates.md).

#### Define the model

```yaml
# resources/models.yaml

- kind: model
  name: dnn
  type: classification
  target: class_indexed
  features:
    - sepal_length_normalized
    - sepal_width_normalized
    - petal_length_normalized
    - petal_width_normalized
  hparams:
    hidden_units: [4, 2]
  data_partition_ratio:
    training: 80
    evaluation: 20
  training:
    num_steps: 1000
    batch_size: 10
```

This configuration will generate a training dataset with the specified features and train our classifier using the generated dataset.

#### Implement the model

Rename `implementations/models/model.py` to `implementations/models/dnn.py` and add TensorFlow code:

```python
# implementations/models/dnn.py

import tensorflow as tf


def create_estimator(run_config, model_config):
    columns = [
        tf.feature_column.numeric_column("sepal_length_normalized"),
        tf.feature_column.numeric_column("sepal_width_normalized"),
        tf.feature_column.numeric_column("petal_length_normalized"),
        tf.feature_column.numeric_column("petal_width_normalized"),
    ]

    return tf.estimator.DNNClassifier(
        feature_columns=columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        n_classes=3,
        config=run_config,
    )
```

Cortex supports any TensorFlow code that adheres to the [tf.estimator API](https://www.tensorflow.org/guide/estimators).

#### Define web APIs

```yaml
# resources/apis.yaml

- kind: api
  name: classifier
  model_name: dnn
  compute:
    replicas: 1
```

This will make the model available as a live web service that can make real-time predictions.

## Deploy the application

```bash
cortex deploy
```

You can track the status of the resources using the `get` and `status` commands:

```bash
cortex get
cortex status
```

You can also view the status of individual resources using the `status` command:

```bash
cortex status class
cortex status dnn
```

#### Test the iris classification service

Rename `samples.json` to `irises.json` and create a sample:

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

Run the prediction:

```bash
cortex predict classifier irises.json
```

This should return "Iris-setosa".

#### Call the API from other clients (e.g. cURL)

Get the API's endpoint:

```bash
cortex get api classifier
```

Use cURL to test the API:

```bash
curl --insecure \
     --request POST \
     --header "Content-Type: application/json" \
     --data '{ "samples": [ { "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 } ] }' \
     https://<ELB name>.us-west-2.elb.amazonaws.com/iris/classifier
```

## Cleanup

Delete the Iris application:

```bash
cortex delete iris
```
