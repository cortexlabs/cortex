<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='88'>

<br>

**Get started:** [Install](https://docs.cortex.dev/install) • [Tutorial](https://docs.cortex.dev/tutorial) • <!-- CORTEX_VERSION_MINOR_STABLE e.g. https://docs.cortex.dev/v/0.2/ -->[Docs](https://docs.cortex.dev) • <!-- CORTEX_VERSION_MINOR_STABLE -->[Examples](https://github.com/cortexlabs/cortex/tree/0.5/examples)

**Learn more:** [Website](https://cortex.dev) • [Blog](https://blog.cortex.dev) • [Subscribe](https://cortexlabs.us20.list-manage.com/subscribe?u=a1987373ab814f20961fd90b4&id=ae83491e1c) • [Twitter](https://twitter.com/cortex_deploy) • [Contact](mailto:hello@cortex.dev)

<br>

Cortex deploys your machine learning models to your cloud infrastructure. You define your deployment with simple declarative configuration, Cortex containerizes your models, deploys them as autoscaling JSON APIs, and manages their lifecycle in production.

Cortex is actively maintained by Cortex Labs. We're a venture-backed team of infrastructure engineers and [we're hiring](https://angel.co/cortex-labs-inc/jobs).

<br>

## How it works

**Define** your deployment using declarative configuration:

```yaml
# cortex.yaml

- kind: api
  name: my-api
  model: s3://my-bucket/my-model.zip
  request_handler: handler.py
  compute:
    min_replicas: 5
    max_replicas: 20
```

**Customize** request handling (optional):

```python
# handler.py

def pre_inference(sample, metadata):
  # Python code


def post_inference(prediction, metadata):
  # Python code
```

**Deploy** to your cloud infrastructure:

```
$ cortex deploy

Deploying ...
Ready! https://amazonaws.com/my-api
```

**Serve** real time predictions via scalable JSON APIs:

```
$ curl -d '{"a": 1, "b": 2, "c": 3}' https://amazonaws.com/my-api

{ prediction: "def" }
```

<br>

## Key features

- **Machine learning deployments as code:** Cortex deployments are defined using declarative configuration.

- **Autoscaling:** Cortex can automatically scale APIs to handle production workloads.

- **Multi framework support:** Cortex supports TensorFlow, Keras, PyTorch, Scikit-learn, XGBoost, and more.

- **CPU / GPU support:** Cortex can run inference on CPU or GPU infrastructure.

- **Rolling updates:** Cortex updates deployed APIs without any downtime.

- **Cloud native:** Cortex can be deployed on any AWS account in minutes.
