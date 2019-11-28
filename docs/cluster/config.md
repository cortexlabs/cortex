# Cluster configuration

The Cortex cluster may be configured by providing a configuration file to `cortex cluster up` or `cortex cluster update` via the  `--config` flag (e.g. `cortex cluster up --config=cluster.yaml`). Below is the schema for the cluster configuration file, with default values shown (unless otherwise specified):

<!-- CORTEX_VERSION_BRANCH_STABLE -->

```yaml
# cluster.yaml

# AWS credentials (if not specified, ~/.aws/credentials will be checked) (can be overridden by $AWS_ACCESS_KEY_ID and $AWS_SECRET_ACCESS_KEY)
aws_access_key_id: ***
aws_secret_access_key: ***

# optional AWS credentials for the operator which may be used to restrict its AWS access (defaults to the AWS credentials set above)
cortex_aws_access_key_id: ***
cortex_aws_secret_access_key: ***

# AWS region
region: us-west-2

# S3 bucket
bucket: cortex-<RANDOM_ID>

# instance type
instance_type: m5.large

# minimum number of instances (must be >= 0)
min_instances: 1

# maximum number of instances (must be >= 1)
max_instances: 5

# instance volume size (GB) (default: 50)
instance_volume_size: 50

# CloudWatch log group for cortex
log_group: cortex

# EKS cluster name for cortex (default: cortex)
cluster_name: cortex

# whether to collect anonymous usage stats and error reports (default: true)
telemetry: true

# whether to use spot instances in the cluster (default: false)
spot: false

spot_config:
  # additional instances with identical or better specs than the primary instance type (defaults to 2 instances sorted by price)
  instance_distribution: [t3.large, t3a.large]  # (defaults for the m5.large instance type)

  # minimum number of on demand instances (default: 0)
  on_demand_base_capacity: 0

  # percentage of on demand instances to use after the on demand base capacity has been met [0, 100] (default: 1)
  # note: setting this to 0 may hinder cluster scale up when spot instances are not available
  on_demand_percentage_above_base_capacity: 1

  # max price for instances (defaults to the on demand price of the primary instance type)
  max_price: 0.096

  # number of spot instance pools across which to allocate spot instances [1, 20] (default: 2)
  instance_pools: 2

# docker image paths
image_predictor_serve: cortexlabs/predictor-serve:0.11.0
image_predictor_serve_gpu: cortexlabs/predictor-serve-gpu:0.11.0
image_tf_serve: cortexlabs/tf-serve:0.11.0
image_tf_serve_gpu: cortexlabs/tf-serve-gpu:0.11.0
image_tf_api: cortexlabs/tf-api:0.11.0
image_onnx_serve: cortexlabs/onnx-serve:0.11.0
image_onnx_serve_gpu: cortexlabs/onnx-serve-gpu:0.11.0
image_operator: cortexlabs/operator:0.11.0
image_manager: cortexlabs/manager:0.11.0
image_downloader: cortexlabs/downloader:0.11.0
image_cluster_autoscaler: cortexlabs/cluster-autoscaler:0.11.0
image_metrics_server: cortexlabs/metrics-server:0.11.0
image_nvidia: cortexlabs/nvidia:0.11.0
image_fluentd: cortexlabs/fluentd:0.11.0
image_statsd: cortexlabs/statsd:0.11.0
image_istio_proxy: cortexlabs/istio-proxy:0.11.0
image_istio_pilot: cortexlabs/istio-pilot:0.11.0
image_istio_citadel: cortexlabs/istio-citadel:0.11.0
image_istio_galley: cortexlabs/istio-galley:0.11.0
```
