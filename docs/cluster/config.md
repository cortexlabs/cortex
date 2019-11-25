# Cluster configuration

The Cortex cluster may be configured by providing a configuration file to `cortex cluster up` or `cortex cluster update` via the  `--config` flag (e.g. `cortex cluster up --config=cluster.yaml`). Below is the schema for the cluster configuration file, with default values shown (unless otherwise specified):

<!-- CORTEX_VERSION_BRANCH_STABLE -->

```yaml
# cluster.yaml

# AWS credentials (if not specified, ~/.aws/credentials will be checked) (can be overridden by $AWS_ACCESS_KEY_ID and $AWS_SECRET_ACCESS_KEY)
aws_access_key_id: ***
aws_secret_access_key: ***

# Optional AWS credentials for the Operator which may be used to restrict its AWS access (defaults to the AWS credentials set above)
cortex_aws_access_key_id: ***
cortex_aws_secret_access_key: ***

# Instance type Cortex will use
instance_type: m5.large

# Minimum number of worker instances in the cluster (must be >= 0)
min_instances: 1

# Maximum number of worker instances in the cluster (must be >= 1)
max_instances: 5

# Volume size in Gi for worker instances in the cluster (default: 50)
instance_volume_size: 50

# Name of the S3 bucket Cortex will use
bucket: cortex-<RANDOM_ID>

# Region Cortex will use
region: us-west-2

# Name of the CloudWatch log group Cortex will use
log_group: cortex

# Name of the EKS cluster Cortex will create
cluster_name: cortex

# Flag to enable collection of anonymous usage stats and error reports
telemetry: true

# Flag to enable using spot instances in worker cluster (default: true)
spot: true

spot_config:
  # List of additional instances with identical or better specs than your primary instance type (defaults to 2 compatible instances sorted by price)
  instance_distribution: [t3.large, t3a.large]

  # The minimum number of instances in your cluster that should be on demand (default: 0)
  on_demand_base_capacity: 0

  # The percentage of on demand instances to use after the on demand base capacity has been met [0, 100] (default: 0)
  on_demand_percentage_above_base_capacity: 0

  # The max price for instances (defaults to the on demand price of the primary instance type)
  max_price: 0.096

  # The number of Spot Instance pools across which to allocate Spot instances [1, 20] (default: 2)
  spot_instance_pools: 2

# Image paths
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
