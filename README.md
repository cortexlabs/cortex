<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='88'>


<br>

**Get Started:** [Install](https://docs.cortexlabs.com/cortex/install) • [Quick Start](https://docs.cortexlabs.com/cortex/quick-start) • [Demo Video](https://www.youtube.com/watch?v=vcistUor0b4) • <!-- CORTEX_VERSION_MINOR_STABLE e.g. https://docs.cortex.dev/v/0.2/ -->[Docs](https://docs.cortex.dev) • <!-- CORTEX_VERSION_MINOR_STABLE -->[Examples](https://github.com/cortexlabs/cortex/tree/0.3/examples)

**Learn More:** [Website](https://cortex.dev) • [FAQ](https://docs.cortexlabs.com/cortex/faq) • [Subscribe](https://cortexlabs.us20.list-manage.com/subscribe?u=a1987373ab814f20961fd90b4&id=ae83491e1c) • [Blog](https://medium.com/cortex-labs) • [Contact](mailto:hello@cortexlabs.com)

<br>

**Machine learning infrastructure for developers:** build and deploy scalable TensorFlow applications on AWS without worrying about setting up infrastructure, managing dependencies, or orchestrating data pipelines.

Cortex is actively maintained by Cortex Labs. We're a venture-backed team of infrastructure engineers and [we're hiring](https://angel.co/cortex-labs-inc/jobs).

<br>

## How it Works

#### Data Validation

```yaml
- kind: raw_column
  name: col1
  type: INT_COLUMN
  min: 0
  max: 10
```

#### Data Ingestion

```yaml
- kind: environment
  name: dev
  data:
    type: csv
    path: s3a://my-data-warehouse/data.csv
    schema: [@col1, @col2, ...]
```

#### Feature Engineering with Python and PySpark

```yaml
- kind: transformed_column
  name: col1_normalized
  transformer_path: normalize.py  # Python / PySpark code
  input: @col1
```

#### Model Training with TensorFlow

```yaml
- kind: model
  name: my_model
  estimator_path: dnn.py  # TensorFlow code
  target_column: @label_col
  input: [@col1_normalized, @col2_indexed, ...]
  hparams:
    hidden_units: [16, 8]
  training:
    batch_size: 32
    num_steps: 10000
```

#### Prediction Serving

```yaml
- kind: api
  name: my-api
  model: @my_model
  compute:
    replicas: 3
```

#### Deploying to AWS

```
$ cortex deploy

Your application is deployed

$ curl -X POST \
       -H "Content-Type: application/json" \
       -d '{ "samples": [ { "col1": 1, "col2": 2, ... } ] }' \
       https://abc.amazonaws.com/my-api

{ "predictions": ["1", "0", ... ] }
```

<br>

## Key Features

- **End-to-end machine learning workflow:** Cortex spans the machine learning workflow from feature management to model training to prediction serving.

- **Machine learning pipelines as code:** Cortex applications are defined using a simple declarative syntax that enables flexibility and reusability.

- **Scalability:** Cortex is designed to handle production workloads.

- **TensorFlow and PySpark support:** Cortex supports custom [TensorFlow](https://www.tensorflow.org) code for model training and custom [PySpark](https://spark.apache.org/docs/latest/api/python/index.html) code for data processing.

- **Built for the cloud:** Cortex can be deployed in any AWS account in minutes.
