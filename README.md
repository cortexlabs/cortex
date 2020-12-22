<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Run inference at scale

Cortex is an open source platform for large-scale inference workloads.

<br>

## Model serving infrastructure

* Supports deploying TensorFlow, PyTorch, sklearn and other models as realtime or batch APIs.
* Ensures high availability with availability zones and automated instance restarts.
* Runs inference on on-demand instances or spot instances with on-demand backups.
* Autoscales to handle production workloads with support for overprovisioning.

#### Configure a cluster

```yaml
# cluster.yaml

region: us-east-1
instance_type: g4dn.xlarge
min_instances: 10
max_instances: 100
spot: true
```

#### Spin up on your AWS or GCP account

```text
$ cortex cluster up --config cluster.yaml

￮ configuring autoscaling ✓
￮ configuring networking ✓
￮ configuring logging ✓

cortex is ready!
```

<br>

## Reproducible deployments

* Package dependencies, code, and configuration for reproducible deployments.
* Configure compute, autoscaling, and networking for each API.
* Integrate with your data science platform or CI/CD system.
* Deploy custom Docker images or use the pre-built defaults.

#### Define an API

```python
class PythonPredictor:
  def __init__(self, config):
    from transformers import pipeline

    self.model = pipeline(task="text-generation")

  def predict(self, payload):
    return self.model(payload["text"])[0]

requirements = ["tensorflow", "transformers"]
```

#### Configure an API

```python
api_spec = {
  "name": "text-generator",
  "kind": "RealtimeAPI",
  "compute": {
    "gpu": 1,
    "mem": "8Gi"
  },
  "autoscaling": {
    "min_replicas": 1,
    "max_replicas": 10
  }
}
```

<br>

## Scalable machine learning APIs

* Scale to handle production workloads with request-based autoscaling.
* Stream performance metrics and logs to any monitoring tool.
* Serve many models efficiently with multi-model caching.
* Use rolling updates to update APIs without downtime.
* Configure traffic splitting for A/B testing.

#### Deploy to your cluster

```python
import cortex

cx = cortex.client("aws")
cx.create_api(api_spec, predictor=PythonPredictor, requirements=requirements)

# creating https://example.com/text-generator
```

#### Consume your API

```bash
$ curl https://example.com/text-generator -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
```

<br>

## Get started

* [Read the docs](https://docs.cortex.dev)
* [Report an issue](https://github.com/cortexlabs/cortex/issues)
* [Join our community](https://gitter.im/cortexlabs/cortex)
