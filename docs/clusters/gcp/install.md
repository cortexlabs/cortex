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
cortex cluster-gcp up cluster.yaml # (see configuration options below)
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

# list of cluster node pools; the smaller index, the higher the priority of the node pool
node_pools:
  - name: np-cpu # name of the node pool
    instance_type: n1-standard-2 # instance type
    # accelerator_type: nvidia-tesla-t4 # GPU to attach to your instance (optional)
    # accelerators_per_instance: 1 # the number of GPUs to attach to each instance (optional)
    min_instances: 1 # minimum number of instances
    max_instances: 5 # maximum number of instances
    preemptible: false  # enable the use of preemptible instances

  - name: np-gpu
    instance_type: n1-standard-2
    accelerator_type: nvidia-tesla-t4
    accelerators_per_instance: 1
    min_instances: 1
    max_instances: 5
    preemptible: false
  ...

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
image_operator: quay.io/cortexlabs/operator:0.31.1
image_manager: quay.io/cortexlabs/manager:0.31.1
image_downloader: quay.io/cortexlabs/downloader:0.31.1
image_request_monitor: quay.io/cortexlabs/request-monitor:0.31.1
image_istio_proxy: quay.io/cortexlabs/istio-proxy:0.31.1
image_istio_pilot: quay.io/cortexlabs/istio-pilot:0.31.1
image_google_pause: quay.io/cortexlabs/google-pause:0.31.1
image_prometheus: quay.io/cortexlabs/prometheus:0.31.1
image_prometheus_config_reloader: quay.io/cortexlabs/prometheus-config-reloader:0.31.1
image_prometheus_operator: quay.io/cortexlabs/prometheus-operator:0.31.1
image_prometheus_statsd_exporter: quay.io/cortexlabs/prometheus-statsd-exporter:0.31.1
image_prometheus_dcgm_exporter: quay.io/cortexlabs/prometheus-dcgm-exporter:0.31.1
image_prometheus_kube_state_metrics: quay.io/cortexlabs/prometheus-kube-state-metrics:0.31.1
image_prometheus_node_exporter: quay.io/cortexlabs/prometheus-node-exporter:0.31.1
image_kube_rbac_proxy: quay.io/cortexlabs/kube-rbac-proxy:0.31.1
image_grafana: quay.io/cortexlabs/grafana:0.31.1
image_event_exporter: quay.io/cortexlabs/event-exporter:0.31.1
```
