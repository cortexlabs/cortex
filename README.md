<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Deploy machine learning models to production

Cortex is an open source platform for deploying, managing, and scaling machine learning in production. Engineering teams use Cortex as an alternative to building in-house machine learning infrastructure.

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->

[install](https://docs.cortex.dev/install) • [documentation](https://docs.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.21/examples) • [support](https://gitter.im/cortexlabs/cortex)

<br>

## Scalable, reliable, and secure model serving infrastructure

* Deploy your TensorFlow, PyTorch, scikit-learn and other models as a realtime or batch APIs
* Scale to handle production workloads with request-based autoscaling
* Configure A/B tests with traffic splitting
* Save money with spot instances

<br>

```yaml
# cluster.yaml

region: us-east-1
api_gateway: public
instance_type: g4dn.xlarge
min_instances: 10
max_instances: 100
spot: true
```

#### Spin up a cluster on your AWS account:

```bash
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

<br>

```yaml
# cortex.yaml

name: image-classifier
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
  endpoint: image-classifier
```

#### Deploy to production:

```bash
$ cortex deploy cortex.yaml

creating https://example.com/image-classifier
```

<br>

## Streamlined machine learning API management

* Monitor API performance
* Aggregate and stream logs
* Customize prediction tracking
* Update APIs without downtime

<br>

```bash
$ cortex get

realtime api       status     replicas   last update   latency   requests

image-classifier   live       10         1h            100ms     100000
object-detector    live       20         2h            200ms     2000000
text-generator     live       30         3h            300ms     30000000


batch api          jobs   id    last update

image-classifier   3      abc   1h
```

<br>

## Get started

```bash
$ pip install cortex
```

See our [installation guide](https://docs.cortex.dev/install) for next steps.
