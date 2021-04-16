<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

[Website](https://www.cortex.dev) • [Slack](https://community.cortex.dev) • [Docs](https://docs.cortex.dev)

# Cost-effective serverless computing at scale

Cortex is a serverless platform for compute-intensive applications.

<br>

## Use cases

* **Data processing** - run image processing, natural language processing, and more.
* **Machine learning in production** - train and serve machine learning models in production.
* **Large-scale inference** - efficiently scale realtime and batch inference workloads.

<br>

## Scalable

* **Cluster autoscaling** - configure Cortex to spin up instances when load increases and spin them down when load decreases.
* **Workload autoscaling** - customize the autoscaling behavior of each workload to ensure efficient use of cluster resources.

```text
$ cortex cluster info

region: us-east-1
instances: [c5.xlarge, g4dn.xlarge]
spot_instances: true
min_instances: 10
max_instances: 100
```

<br>

## Flexible

* **Any workload** - define custom Python functions or containers and deploy them as realtime, async, and batch workloads.
* **Any pipeline** - chain workloads seamlessly to create custom data pipelines.

```text
$ cortex deploy apis.yaml

creating text-generator (realtime API)
creating image-classifier (batch API)
creating video-analyzer (async API)

all APIs are ready!
```

<br>

## Observable

* **Structured logging** - automatically stream logs from every workload to your favorite log management tool.
* **Workload autoscaling** - monitor your workloads with pre-built Grafana dashboards and add your own custom dashboards.

```text
$ cortex get

API                TYPE       REPLICAS
text-generator     realtime   32
image-classifier   batch      64
video-analyzer     async      16
```

<br>

## Affordable

* **Spot instance management** - spend less on EC2 instances by letting Cortex manage spot instances.
* **Multi-instance type clusters** - configure resources per workload to run each workload on the right hardware.

```text
$ cortex cluster pricing

RESOURCE                       COST PER HOUR
1 eks cluster                  $0.10
2 network load balancers       $0.02 each
10-100 g4dn.xlarge instances   $0.53 each
10-100 c5.xlarge instances     $0.17 each
```
