<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='88'>

<br>

**Get started:** [Install](https://docs.cortex.dev/install) • [Tutorial](https://docs.cortex.dev/tutorial) • [Demo Video](https://www.youtube.com/watch?v=vcistUor0b4) • <!-- CORTEX_VERSION_MINOR_STABLE e.g. https://docs.cortex.dev/v/0.2/ -->[Docs](https://docs.cortex.dev) • <!-- CORTEX_VERSION_MINOR_STABLE -->[Examples](https://github.com/cortexlabs/cortex/tree/0.4/examples)

**Learn more:** [Website](https://cortex.dev) • [FAQ](https://docs.cortex.dev/faq) • [Blog](https://blog.cortex.dev) • [Subscribe](https://cortexlabs.us20.list-manage.com/subscribe?u=a1987373ab814f20961fd90b4&id=ae83491e1c) • [Twitter](https://twitter.com/cortex_deploy) • [Contact](mailto:hello@cortex.dev)

<br>

## Deploy, manage, and scale machine learning applications

Deploy machine learning applications without worrying about setting up infrastructure, managing dependencies, or orchestrating data pipelines.

Cortex is actively maintained by Cortex Labs. We're a venture-backed team of infrastructure engineers and [we're hiring](https://angel.co/cortex-labs-inc/jobs).

<br>

## How it works

1. **Define your app:** define your app using Python, TensorFlow, and PySpark.

2. **`$ cortex deploy`:** deploy end-to-end machine learning pipelines to AWS with one command.

3. **Serve predictions:** serve real time predictions via horizontally scalable JSON APIs.

<br>

## End-to-end machine learning workflow

**Data ingestion:** connect to your data warehouse and ingest data.

```yaml
- kind: environment
  name: dev
  data:
    type: csv
    path: s3a://my-bucket/data.csv
    schema: [@col1, @col2, ...]
```

**Data validation:** prevent data quality issues early.

```yaml
- kind: raw_column
  name: col1
  type: INT_COLUMN
  min: 0
  max: 10
```

**Data transformation:** use custom Python and PySpark code to transform data.

```yaml
- kind: transformed_column
  name: col1_normalized
  transformer_path: normalize.py  # Python / PySpark code
  input: @col1
```

**Model training:** train models with custom TensorFlow code.

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

**Prediction serving:** serve real time predictions via JSON APIs.

```yaml
- kind: api
  name: my-api
  model: @my_model
  compute:
    replicas: 3
```

**Deployment:** Cortex deploys your pipeline on scalable cloud infrastructure.

```
$ cortex deploy
Ingesting data ...
Transforming data ...
Training models ...
Deploying API ...
Ready! https://abc.amazonaws.com/my-api
```

<br>

## Key features

- **Machine learning pipelines as code:** Cortex applications are defined using a simple declarative syntax that enables flexibility and reusability.

- **End-to-end machine learning workflow:** Cortex spans the machine learning workflow from feature management to model training to prediction serving.

- **TensorFlow and PySpark support:** Cortex supports custom [TensorFlow](https://www.tensorflow.org) code for model training and custom [PySpark](https://spark.apache.org/docs/latest/api/python/index.html) code for data processing.

- **Built for the cloud:** Cortex can handle production workloads and can be deployed in any AWS account in minutes.
