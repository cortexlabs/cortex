<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='88'>

<br>

**Get started:** [Install](https://docs.cortex.dev/install) • [Tutorial](https://docs.cortex.dev/tutorial) • [Demo Video](https://www.youtube.com/watch?v=tgMjCOD_ufo) • <!-- CORTEX_VERSION_MINOR_STABLE e.g. https://docs.cortex.dev/v/0.2/ -->[Docs](https://docs.cortex.dev) • <!-- CORTEX_VERSION_MINOR_STABLE -->[Examples](https://github.com/cortexlabs/cortex/tree/0.4/examples)

**Learn more:** [Website](https://cortex.dev) • [Blog](https://blog.cortex.dev) • [Subscribe](https://cortexlabs.us20.list-manage.com/subscribe?u=a1987373ab814f20961fd90b4&id=ae83491e1c) • [Twitter](https://twitter.com/cortex_deploy) • [Contact](mailto:hello@cortex.dev)

<br>

## Deploy machine learning models in production

Cortex deploys your machine learning models on your cloud infrastructure. You define your deployment with simple declarative configuration, Cortex containerizes your models, deploys them as scalable JSON APIs, and manages their lifecycle in production.

Cortex is actively maintained by Cortex Labs. We're a venture-backed team of infrastructure engineers and [we're hiring](https://angel.co/cortex-labs-inc/jobs).

<br>

## Machine learning deployments as code

**Configure:** Define your deployment using simple declarative configuration.

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/my-model.zip    # TensorFlow / PyTorch
  preprocessor: transform_payload.py    # Transform request payloads before inference
  postprocessor: process_prediction.py  # Transform predicitons before responding to the client
  compute:
    replicas: 3
    gpu: 2
```

**Deploy:** Cortex deploys your pipeline on scalable cloud infrastructure.

```
$ cortex deploy

Provisioning infrastructure ...
Deploying API ...
Loading model ...

Ready! https://amazonaws.com/my-api
```

**Manage:** Serve real time predictions via scalable JSON APIs.

```
$ cortex status my-api

Endpoint: https://amazonaws.com/my-api
Latency: 200ms
Throughput: 50 requests per second
Predictions: 1,234,567 legitimate | 89 fraud
```

<br>

## Key features

- **Machine learning pipelines as code:** Cortex deployments are defined using a simple declarative syntax that enables flexibility and reusability.

- **Multi framework support:** Cortex supports [TensorFlow](https://www.tensorflow.org), [Keras](https://keras.io), and [PyTorch](https://pytorch.org) models.

- **Scalability:** Cortex automatically handles scaling APIs.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **A/B testing:** Cortex can load balance traffic across multiple models.

- **Cloud agnostic:** Cortex can handle production workloads and can be deployed on any Kubernetes cluster in minutes.
