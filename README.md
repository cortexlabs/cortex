<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Deploy machine learning in production

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->

We are building Cortex to help machine learning engineers and data scientists deploy, manage, and scale machine learning in production. Organizations worldwide use Cortex as an alternative to building in-house machine learning infrastructure. You can [get started now](https://docs.cortex.dev/install) or learn more by checking out our [docs](https://docs.cortex.dev) and [examples](https://github.com/cortexlabs/cortex/tree/0.20/examples). If you have any questions or feedback, we'd love to hear from you on our [Gitter](https://gitter.im/cortexlabs/cortex)!

<br>

```bash
$ pip install cortex
```

<br>

## Scalable, reliable, and secure model serving infrastructure

* **Deploy any machine learning model as a realtime or batch API**
* **Scale to handle production workloads with request-based autoscaling**
* **Configure A/B tests with traffic splitting**
* **Save money with spot instances**

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

**Spin up a cluster on your AWS account**

```bash
$ cortex cluster up --config cluster.yaml

￮ configuring autoscaling ✓
￮ configuring networking ✓
￮ configuring logging ✓

cortex is ready!
```

<br>

## Simple, flexible, and reproducible deployments

* **Deploy your TensorFlow, PyTorch, ONNX, and other models**
* **Customize compute, autoscaling, and networking for each API**
* **Manage dependencies for reproducible deployments**
* **Define request handling in Python**
* **Iterate on deployments in local mode**

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

**Deploy machine learning in production**

```bash
$ cortex deploy --env aws
```

<br>

## One tool for managing machine learning APIs

* **Monitor API performance**
* **Aggregate and stream logs**
* **Update APIs without any downtime**

```bash
$ cortex get

api                status     replicas   last update   latency   requests

text-generator     live       10         1h            100ms     10000000
image-classifier   live       20         2h            200ms     20000000
object-detector    live       30         3h            300ms     30000000
```
