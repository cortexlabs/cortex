<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Deploy machine learning models to production

Cortex is an open source platform for deploying, managing, and scaling machine learning in production. Organizations worldwide use Cortex as an alternative to building in-house machine learning infrastructure.

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->

[install](https://docs.cortex.dev/install) • [documentation](https://docs.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.21/examples) • [support](https://gitter.im/cortexlabs/cortex)

<br>

## Scalable, reliable, and secure model serving infrastructure

* Deploy any machine learning model as a realtime or batch API
* Scale to handle production workloads with request-based autoscaling
* Configure A/B tests with traffic splitting
* Save money with spot instances

<br>

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

### Spin up a cluster on your AWS account

```bash
$ cortex cluster up --config cluster.yaml

￮ configuring autoscaling ✓
￮ configuring networking ✓
￮ configuring logging ✓

cortex is ready!
```

<br>

## Simple, flexible, and reproducible deployments

* Deploy your TensorFlow, PyTorch, ONNX, and other models
* Define request handling in Python
* Test deployments locally before deploying to production
* Customize compute, autoscaling, and networking for each API
* Package dependencies, code, and configuration for reproducible deployments

<br>

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
  endpoint: my-api
```

### Deploy a model to production

```bash
$ cortex deploy --env aws

creating https://example.com/text-generator
```

<br>

## Streamlined machine learning API management

* Monitor API performance
* Aggregate and stream logs
* Customize prediction tracking
* Update APIs without any downtime

<br>

```bash
$ cortex get

api                status     replicas   last update   latency   requests

text-generator     live       10         1h            100ms     100000
image-classifier   live       20         2h            200ms     2000000
object-detector    live       30         3h            300ms     30000000
```

<br>

## Get started

```bash
$ pip install cortex
```
