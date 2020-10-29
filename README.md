<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->

[install](https://docs.cortex.dev/install) • [documentation](https://docs.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.21/examples) • [support](https://gitter.im/cortexlabs/cortex)

# Deploy machine learning models to production

Cortex is an open source platform for deploying, managing, and scaling machine learning in production. Engineering teams use Cortex as an alternative to building in-house machine learning infrastructure.

<br>

## Scalable, reliable, and secure model serving infrastructure

* Deploy your TensorFlow, PyTorch, scikit-learn and other models as a realtime or batch APIs
* Scale to handle production workloads with request-based autoscaling
* Run on inference on spot instances with on demand backups
* Configure A/B tests with traffic splitting

#### Configure your cluster:

```yaml
# cluster.yaml

region: us-east-1
api_gateway: public
instance_type: g4dn.xlarge
min_instances: 10
max_instances: 100
spot: true
```

#### Spin up your cluster on your AWS account:

```text
$ cortex cluster up --config cluster.yaml

￮ configuring autoscaling ✓
￮ configuring networking ✓
￮ configuring logging ✓

cortex is ready!
```

<br>

## Simple, flexible, and reproducible deployments

* Implement request handling in Python
* Customize compute, autoscaling, and networking for each API
* Package dependencies, code, and configuration for reproducible deployments
* Test locally before deploying to production

#### Implement a predictor:

```python
# predictor.py

from transformers import pipeline

class PythonPredictor:
  def __init__(self, config):
    self.model = pipeline(task="text-generation")

  def predict(self, payload):
    return self.model(payload["text"])[0]
```

#### Configure a realtime API:

```yaml
# cortex.yaml

name: text-generator
kind: RealtimeAPI
predictor:
  path: predictor.py
compute:
  gpu: 1
  mem: 4Gi
autoscaling:
  min_replicas: 1
  max_replicas: 10
networking:
  endpoint: text-generator
```

#### Deploy to production:

```text
$ cortex deploy cortex.yaml

creating https://example.com/text-generator
```

<br>

## Streamlined machine learning API management

* Monitor API performance
* Aggregate and stream logs
* Customize prediction tracking
* Update APIs without downtime

#### Manage your APIs:

```text
$ cortex get

realtime api       status     replicas   last update   latency   requests

image-classifier   live       5          1h            100ms     100000
object-detector    live       10         2h            200ms     1000000
text-generator     live       20         3h            300ms     10000000


batch api          jobs   last update

image-classifier   3      1h
```

<br>

## Get started

```text
$ pip install cortex
```

See the [installation guide](https://docs.cortex.dev/install) for next steps.
