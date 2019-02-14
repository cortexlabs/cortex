# Quick Start

## Prerequisites

1. An AWS account
2. A Kubernetes cluster running the Cortex operator ([installation instructions](operator/install.md))
3. The Cortex CLI

## TL;DR

You can download pre-built applications from our repository:

<!-- CORTEX_VERSION_MINOR -->

```bash
git clone -b master https://github.com/cortexlabs/cortex.git
cd cortex/examples/iris
```

Jump to [deploy the application](#deploy-the-application).

## Build a machine learning application

Let's build and deploy a classifier using the famous [Iris Data Set](https://archive.ics.uci.edu/ml/datasets/iris)! Below are a few samples of iris data:

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
    schema:
      - sepal_length
      - sepal_width
      - petal_length
      - petal_width
      - class
```

Cortex will be able to read from any S3 bucket that your AWS credentials grant access to.

#### Define raw columns

The Iris Data Set consists of four attributes and a label. We ensure that the data matches the types we expect, the numerical data is within a reasonable range, and the class labels are within the set of expected labels.

Add to `app.yaml`:

```yaml
# Raw Columns

- kind: raw_column
  name: sepal_length
  type: FLOAT_COLUMN
  min: 0
  max: 10

- kind: raw_column
  name: sepal_width
  type: FLOAT_COLUMN
  min: 0
  max: 10

- kind: raw_column
  name: petal_length
  type: FLOAT_COLUMN
  min: 0
  max: 10

- kind: raw_column
  name: petal_width
  type: FLOAT_COLUMN
  min: 0
  max: 10

- kind: raw_column
  name: class
  type: STRING_COLUMN
  values: ['Iris-setosa', 'Iris-versicolor', 'Iris-virginica']
```

#### Define aggregates

Aggregates are computations that require processing a full column of data. We want to normalize the numeric columns, so we need mean and standard deviation values for each numeric column. We also need a mapping of strings to integers for the label column. Here we use the built-in `mean`, `stddev`, and `index_string` aggregators.

Add to `app.yaml`:

```yaml
# Aggregates

- kind: aggregate
  name: sepal_length_mean
  aggregator: cortex.mean
  inputs:
    columns:
      col: sepal_length

- kind: aggregate
  name: sepal_length_stddev
  aggregator: cortex.stddev
  inputs:
    columns:
      col: sepal_length

- kind: aggregate
  name: sepal_width_mean
  aggregator: cortex.mean
  inputs:
    columns:
      col: sepal_width

- kind: aggregate
  name: sepal_width_stddev
  aggregator: cortex.stddev
  inputs:
    columns:
      col: sepal_width

- kind: aggregate
  name: petal_length_mean
  aggregator: cortex.mean
  inputs:
    columns:
      col: petal_length

- kind: aggregate
  name: petal_length_stddev
  aggregator: cortex.stddev
  inputs:
    columns:
      col: petal_length

- kind: aggregate
  name: petal_width_mean
  aggregator: cortex.mean
  inputs:
    columns:
      col: petal_width

- kind: aggregate
  name: petal_width_stddev
  aggregator: cortex.stddev
  inputs:
    columns:
      col: petal_width

- kind: aggregate
  name: class_index
  aggregator: cortex.index_string
  inputs:
    columns:
      col: class
```

#### Define transformed columns

Transformers convert the raw columns into the appropriate inputs for a TensorFlow model. Here we use the built-in `normalize` and `index_string` transformers using the aggregates we computed earlier.

Add to `app.yaml`:

```yaml
# Transformed Columns

- kind: transformed_column
  name: sepal_length_normalized
  transformer: cortex.normalize
  inputs:
    columns:
      num: sepal_length
    args:
      mean: sepal_length_mean
      stddev: sepal_length_stddev

- kind: transformed_column
  name: sepal_width_normalized
  transformer: cortex.normalize
  inputs:
    columns:
      num: sepal_width
    args:
      mean: sepal_width_mean
      stddev: sepal_width_stddev

- kind: transformed_column
  name: petal_length_normalized
  transformer: cortex.normalize
  inputs:
    columns:
      num: petal_length
    args:
      mean: petal_length_mean
      stddev: petal_length_stddev

- kind: transformed_column
  name: petal_width_normalized
  transformer: cortex.normalize
  inputs:
    columns:
      num: petal_width
    args:
      mean: petal_width_mean
      stddev: petal_width_stddev

- kind: transformed_column
  name: class_indexed
  transformer: cortex.index_string
  inputs:
    columns:
      text: class
    args:
      index: class_index
```

You can simplify the YAML for aggregates and transformed columns using [templates](applications/advanced/templates.md).

#### Define the model

This configuration will generate a training dataset with the specified columns and train our classifier using the generated dataset.

Add to `app.yaml`:

```yaml
# Models

- kind: model
  name: dnn
  path: dnn.py
  type: classification
  target_column: class_indexed
  feature_columns:
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

#### Implement the estimator

Define an estimator in `dnn.py`:

```python
import tensorflow as tf


def create_estimator(run_config, model_config):
    feature_columns = [
        tf.feature_column.numeric_column("sepal_length_normalized"),
        tf.feature_column.numeric_column("sepal_width_normalized"),
        tf.feature_column.numeric_column("petal_length_normalized"),
        tf.feature_column.numeric_column("petal_width_normalized"),
    ]

    # returns an instance of tf.estimator.Estimator
    return tf.estimator.DNNClassifier(
        feature_columns=feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        n_classes=3,
        config=run_config,
    )
```

Cortex supports any TensorFlow code that adheres to the [tf.estimator API](https://www.tensorflow.org/guide/estimators).

#### Define web APIs

This will make the model available as a live web service that can make real-time predictions.

Add to `app.yaml`:

```yaml
# APIs

- kind: api
  name: iris-type
  model_name: dnn
  compute:
    replicas: 1
```

## Deploy the application

```
$ cortex deploy

Deployment started
```

The first deployment may take some extra time as Cortex's dependencies are downloaded.

You can get an overview of the deployment using `cortex get` (see [resource statuses](applications/resources/statuses.md) for the meaning of each status):

```
$ cortex get

-----------
Raw Columns
-----------

NAME                               STATUS                  AGE
class                              ready                   56s
petal_length                       ready                   56s
petal_width                        ready                   56s
sepal_length                       ready                   56s
sepal_width                        ready                   56s

----------
Aggregates
----------

NAME                               STATUS                  AGE
class_index                        ready                   33s
petal_length_mean                  ready                   44s
petal_length_stddev                ready                   44s
petal_width_mean                   ready                   44s
petal_width_stddev                 ready                   44s
sepal_length_mean                  ready                   44s
sepal_length_stddev                ready                   44s
sepal_width_mean                   ready                   44s
sepal_width_stddev                 ready                   44s

-------------------
Transformed Columns
-------------------

NAME                               STATUS                  AGE
class_indexed                      ready                   29s
petal_length_normalized            ready                   26s
petal_width_normalized             ready                   23s
sepal_length_normalized            ready                   20s
sepal_width_normalized             ready                   17s

-----------------
Training Datasets
-----------------

NAME                               STATUS                  AGE
dnn/training_dataset               ready                   9s

------
Models
------

NAME                               STATUS                  AGE
dnn                                training                -

----
APIs
----

NAME                               STATUS                  LAST UPDATE
iris-type                          pending                 -
```

You can get a summary of the status of resources using `cortex status`:

```
$ cortex status --watch

Raw Columns:           5 ready
Aggregates:            9 ready
Transformed Columns:   5 ready
Training Datasets:     1 ready
Models:                1 training
APIs:                  1 pending
```

You can also view the status of individual resources using the `status` command:

```
$ cortex status sepal_length_normalized

---------
Ingesting
---------

Ingesting iris data from s3a://cortex-examples/iris.csv
Caching iris data (version: 2019-02-12-22-55-07-611766)
150 rows ingested

Reading iris data (version: 2019-02-12-22-55-07-611766)

First 3 samples:

class:         Iris-setosa    Iris-setosa    Iris-setosa
petal_length:  1.40           1.40           1.30
petal_width:   0.20           0.20           0.20
sepal_length:  5.10           4.90           4.70
sepal_width:   3.50           3.00           3.20

-----------
Aggregating
-----------

Aggregating petal_length_mean
Aggregating petal_length_stddev
Aggregating petal_width_mean
Aggregating petal_width_stddev
Aggregating sepal_length_mean
Aggregating sepal_length_stddev
Aggregating sepal_width_mean
Aggregating sepal_width_stddev
Aggregating class_index

Aggregates:

class_index:          ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]
petal_length_mean:    3.7586666552225747
petal_length_stddev:  1.7644204144315179
petal_width_mean:     1.198666658103466
petal_width_stddev:   0.7631607319020202
sepal_length_mean:    5.843333326975505
sepal_length_stddev:  0.8280661128539085
sepal_width_mean:     3.0540000025431313
sepal_width_stddev:   0.43359431104332985

-----------------------
Validating Transformers
-----------------------

Sanity checking transformers against the first 100 samples

Transforming class to class_indexed
class:          Iris-setosa    Iris-setosa    Iris-setosa
class_indexed:  0              0              0

Transforming petal_length to petal_length_normalized
petal_length:           1.40     1.40     1.30
petal_length_norm...:  -1.34    -1.34    -1.39

Transforming petal_width to petal_width_normalized
petal_width:            0.20     0.20     0.20
petal_width_norma...:  -1.31    -1.31    -1.31

Transforming sepal_length to sepal_length_normalized
sepal_length:           5.10     4.90     4.70
sepal_length_norm...:  -0.90    -1.14    -1.38

Transforming sepal_width to sepal_width_normalized
sepal_width:           3.50     3.00    3.20
sepal_width_norma...:  1.03    -0.12    0.34

----------------------------
Generating Training Datasets
----------------------------

Generating dnn/training_dataset

Completed on Tuesday, February 14, 2019 at 2:56pm PST
```

```
$ cortex status dnn

--------
Training
--------

loss = 13.321785, step = 1
loss = 3.8588388, step = 101 (0.226 sec)
loss = 4.1841183, step = 201 (0.241 sec)
loss = 4.089279, step = 301 (0.194 sec)
loss = 1.646344, step = 401 (0.174 sec)
loss = 2.367354, step = 501 (0.189 sec)
loss = 2.0011806, step = 601 (0.192 sec)
loss = 1.7621514, step = 701 (0.211 sec)
loss = 0.8322474, step = 801 (0.190 sec)
loss = 1.3244338, step = 901 (0.194 sec)

----------
Evaluating
----------

accuracy = 0.96153843
average_loss = 0.13040856
global_step = 1000
loss = 3.3906221

-------
Caching
-------

Caching model dnn

Completed on Tuesday, February 14, 2019 at 2:56pm PST
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

When the API is ready, request a prediction from the API like so:

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

-------
Summary
-------

Endpoint:         https://a84607a462f1811e9aa3b020abd0a844-645332984.us-west-2.elb.amazonaws.com/iris/iris-type
Status:           ready
Created at:       2019-02-14 14:57:04 PST
Refreshed at:     2019-02-14 14:57:35 PST
Updated replicas: 1/1 ready

-------------
Configuration
-------------

{
 "name": "iris-type",
 "model_name": "dnn",
 "compute": {
   "replicas": 1,
   "cpu": <null>,
   "mem": <null>
 },
 "tags": {}
}
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
