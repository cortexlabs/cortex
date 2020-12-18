# Install

## Spin up Cortex on your GCP account

Make sure [Docker](https://docs.docker.com/install) is running on your machine.

If you're using GPUs, make sure your GPU quota is sufficient (see [here](https://cloud.google.com/compute/quotas)).

```bash
# install the CLI
pip install cortex

# spin up Cortex on your GCP account
cortex cluster-gcp up  # or: cortex cluster-gcp up --config cluster.yaml (see configuration options below)

# set the default environment
cortex env default gcp
```

## Configure Cortex

```yaml
# cluster.yaml

# GKE cluster name
cluster_name: cortex

# GCP project ID
project: <your GCP project ID>

# GCP zone for your cluster
zone: us-central1-a

# instance type
instance_type: n1-standard-2

# GPU to attach to your instance (optional)
accelerator_type: nvidia-tesla-t4

# minimum number of instances
min_instances: 1

# maximum number of instances
max_instances: 5
```

The docker images used by the Cortex cluster can also be overridden, although this is not common. They can be configured by adding any of these keys to your cluster configuration file (default values are shown):

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```yaml
image_operator: quay.io/cortexlabs/operator:master
image_manager: quay.io/cortexlabs/manager:master
image_downloader: quay.io/cortexlabs/downloader:master
image_statsd: quay.io/cortexlabs/statsd:master
image_istio_proxy: quay.io/cortexlabs/istio-proxy:master
image_istio_pilot: quay.io/cortexlabs/istio-pilot:master
image_pause: quay.io/cortexlabs/pause:master
```
