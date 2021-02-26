# Install

## Prerequisites

1. [Docker](https://docs.docker.com/install) must be installed and running on your machine (to verify, check that running `docker ps` does not return an error)
1. You may need to [request a quota increase](https://cloud.google.com/compute/quotas) for your desired instance type and/or GPU type
1. Export the `GOOGLE_APPLICATION_CREDENTIALS` environment variable, containing the path to your GCP credentials file (e.g. `export GOOGLE_APPLICATION_CREDENTIALS=~/.config/gcloud/myproject-8a41417a968a.json`)
1. If you haven't done so already, enable the Kubernetes Engine API in your GCP project ([here](https://console.developers.google.com/apis/api/container.googleapis.com/overview))

## Spin up Cortex on your GCP account

```bash
# install the CLI
pip install cortex

# spin up Cortex on your GCP account
cortex cluster-gcp up  # or: cortex cluster-gcp up --config cluster.yaml (see configuration options below)
```

## Configure Cortex

```yaml
# cluster.yaml

# GKE cluster name
cluster_name: cortex

# GCP project ID
project: <your GCP project ID>

# GCP zone for your cluster
zone: us-east1-c

# instance type
instance_type: n1-standard-2

# minimum number of instances
min_instances: 1

# maximum number of instances
max_instances: 5

# enable the use of preemptible instances
preemptible: false

# enable the use of on-demand backup instances which will be used when preemptible capacity runs out
# default is true when preemptible instances are used
# on_demand_backup: true

# GPU to attach to your instance (optional)
# accelerator_type: nvidia-tesla-t4

# the number of GPUs to attach to each instance (optional)
# accelerators_per_instance: 1

# the name of the network in which to create your cluster
# network: default

# the name of the subnetwork in which to create your cluster
# subnet: default

# API load balancer scheme [internet-facing | internal]
api_load_balancer_scheme: internet-facing

# operator load balancer scheme [internet-facing | internal]
# note: if using "internal", you must be within the cluster's VPC or configure VPC Peering to connect your CLI to your cluster operator
operator_load_balancer_scheme: internet-facing
```

The docker images used by the Cortex cluster can also be overridden, although this is not common. They can be configured by adding any of these keys to your cluster configuration file (default values are shown):

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```yaml
image_operator: quay.io/cortexlabs/operator:master
image_manager: quay.io/cortexlabs/manager:master
image_downloader: quay.io/cortexlabs/downloader:master
image_request_monitor: quay.io/cortexlabs/request-monitor:master
image_istio_proxy: quay.io/cortexlabs/istio-proxy:master
image_istio_pilot: quay.io/cortexlabs/istio-pilot:master
image_google_pause: quay.io/cortexlabs/google-pause:master
image_prometheus: quay.io/cortexlabs/prometheus:master
image_prometheus_config_reloader: quay.io/cortexlabs/prometheus-config-reloader:master
image_prometheus_operator: quay.io/cortexlabs/prometheus-operator:master
image_prometheus_statsd_exporter: quay.io/cortexlabs/prometheus-statsd-exporter:master
image_grafana: quay.io/cortexlabs/grafana:master
image_event_exporter: quay.io/cortexlabs/event-exporter:master
```
