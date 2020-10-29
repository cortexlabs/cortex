<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->

[install](https://docs.cortex.dev/install) • [documentation](https://docs.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.21/examples) • [support](https://gitter.im/cortexlabs/cortex)

# Deploy machine learning models to production

Cortex is an open source platform for deploying, managing, and scaling machine learning in production. Engineering teams use Cortex as an alternative to building in-house machine learning infrastructure.

<br>

## Model serving infrastructure

* Supports deploying TensorFlow, PyTorch, sklearn and other models as realtime or batch APIs
* Ensures high availability with availability zones and automated instance restarts
* Scales to handle production workloads with request-based autoscaling
* Runs inference on spot instances with on-demand backups
* Manages traffic splitting for A/B testing

#### Configure your cluster:

```yaml
# cluster.yaml

region: us-east-1
availability_zones: [us-east-1a, us-east-1b]
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
￮ configuring metrics dashboard ✓

cortex is ready!
```

<br>

## Reproducible model deployments

* Implement request handling in Python
* Customize compute, autoscaling, and networking for each API
* Package dependencies, code, and configuration for reproducible deployments
* Test locally before deploying to your cluster

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

#### Configure an API:

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
  api_gateway: public
```

#### Deploy to production:

```text
$ cortex deploy cortex.yaml

creating https://example.com/text-generator

$ curl https://example.com/text-generator \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "deploy machine learning models in"}'

"deploy machine learning models in production"
```

<br>

## API management

* Monitor API performance
* Aggregate and stream logs
* Customize prediction tracking
* Update APIs without downtime

#### Manage your APIs:

```text
$ cortex get

realtime api       status     replicas   last update   latency   requests

text-generator     live       34         9h            247ms     71828
object-detector    live       13         15h           23ms      828459
image-classifier   live       5          3d 14h        88ms      4523


batch api          running jobs   last update

image-classifier   3              10h
```

<br>

## Get started

```text
$ pip install cortex
```

See the [installation guide](https://docs.cortex.dev/install) for next steps.
