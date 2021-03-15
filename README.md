<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

[Website](https://www.cortex.dev) • [Slack](https://community.cortex.dev) • [Docs](https://docs.cortex.dev)

<br>

# Deploy, manage, and scale machine learning models in production

Cortex is a cloud native model serving platform for machine learning engineering teams.

<br>

## Use cases

* **Realtime machine learning** - build NLP, computer vision, and other APIs and integrate them into any application.
* **Large-scale inference** - scale realtime or batch inference workloads across hundreds or thousands of instances.
* **Consistent MLOps workflows** - create streamlined and reproducible MLOps workflows for any machine learning team.

<br>

## Deploy

* Deploy TensorFlow, PyTorch, ONNX, and other models using a simple CLI or Python client.
* Run realtime inference, batch inference, asynchronous inference, and training jobs.
* Define preprocessing and postprocessing steps in Python and chain workloads seamlessly.

```text
$ cortex deploy apis.yaml

• creating text-generator (realtime API)
• creating image-classifier (batch API)
• creating video-analyzer (async API)

all APIs are ready!
```

## Manage

* Create A/B tests and shadow pipelines with configurable traffic splitting.
* Automatically stream logs from every workload to your favorite log management tool.
* Monitor your workloads with pre-built Grafana dashboards and add your own custom dashboards.

```text
$ cortex get

API                 TYPE        GPUs
text-generator      realtime    32
image-classifier    batch       64
video-analyzer      async       16
```

## Scale

* Configure workload and cluster autoscaling to efficiently handle large-scale production workloads.
* Create clusters with different types of instances for different types of workloads.
* Spend less on cloud infrastructure by letting Cortex manage spot or preemptible instances.

```text
$ cortex cluster info

provider: aws
region: us-east-1
instance_types: [c5.xlarge, g4dn.xlarge]
spot_instances: true
min_instances: 10
max_instances: 100
```
