<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Deploy machine learning to production

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->

Cortex is an open source platform for deploying, managing, and scaling machine learning in production. Organizations worldwide use Cortex as an alternative to building in-house machine learning infrastructure. You can [get started now](https://docs.cortex.dev/install) or learn more by checking out our [docs](https://docs.cortex.dev) and [examples](https://github.com/cortexlabs/cortex/tree/0.20/examples). If you have any questions or feedback, we'd love to hear from you on our [Gitter](https://gitter.im/cortexlabs/cortex)!

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
* **Run the Cortex cluster on your AWS account**


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
<img src="https://uploads-ssl.webflow.com/5f6030ed637ab64288922cff/5f6bc7244d5a3d5f630e78f7_carbon%20(10).png" height="200">

<br>

## Simple, flexible, and reproducible deployments

* **Deploy models from TensorFlow, PyTorch, ONNX, or any other Python framework**
* **Customize compute, autoscaling, and networking for each API**
* **Define inference logic in Python**
* **Package dependencies, code, and configuration for reproducible deployments**
* **Test deployments locally before deploying to production**
* **Import any model server, realtime feature store, library, or tool**

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

<img src="https://uploads-ssl.webflow.com/5f6030ed637ab64288922cff/5f6ba3b8a56985d25e9dfe8b_deploy-cli.png" height="200">
<br>

## One tool for managing machine learning APIs

* **Monitor API performance**
* **Aggregate and stream logs**
* **Customize prediction tracking**
* **Exoprt performance data to any analytics tool**
* **Update APIs with zero downtime**

<img src="https://uploads-ssl.webflow.com/5f6030ed637ab64288922cff/5f99ea232813f90e78bac7a5_carbon%20(33).png" height="200">
<br>

# Get started
You can get started by [installing Cortex](https://docs.cortex.dev) and following our [quickstart tutorial](https://docs.cortex.dev/v/0.21/deployments/realtime-api/text-generator). Additionally, you can test Cortex out by deploying any of our [example APIs.](https://github.com/cortexlabs/cortex/tree/master/examples)
