# Cluster configuration

The Cortex cluster may be configured by providing a configuration file to `cortex cluster up` or `cortex cluster update` via the `--config` flag (e.g. `cortex cluster up --config=cluster.yaml`). Below is the schema for the cluster configuration file, with default values shown (unless otherwise specified):

<!-- CORTEX_VERSION_MINOR -->
```yaml
# cluster.yaml

# AWS credentials (if not specified, ~/.aws/credentials will be checked) (can be overridden by $AWS_ACCESS_KEY_ID and $AWS_SECRET_ACCESS_KEY)
aws_access_key_id: ***
aws_secret_access_key: ***

# optional AWS credentials for the operator which may be used to restrict its AWS access (defaults to the AWS credentials set above)
cortex_aws_access_key_id: ***
cortex_aws_secret_access_key: ***

# EKS cluster name for cortex (default: cortex)
cluster_name: cortex

# AWS region
region: us-west-2

# S3 bucket (default: <cluster_name>-<RANDOM_ID>)
bucket: # cortex-<RANDOM_ID>

# List of availability zones for your region (default: 3 random availability zones from the specified region)
availability_zones: # e.g. [us-west-2a, us-west-2b, us-west-2c]

# instance type
instance_type: m5.large

# minimum number of instances (must be >= 0)
min_instances: 1

# maximum number of instances (must be >= 1)
max_instances: 5

# instance volume size (GB) (default: 50)
instance_volume_size: 50

# CloudWatch log group for cortex (default: <cluster_name>)
log_group: cortex

# whether to use spot instances in the cluster (default: false)
# see https://cortex.dev/v/0.15/cluster-management/spot-instances for additional details on spot configuration
spot: false
```

The docker images used by Cortex are listed below. They can be overridden to use custom images by specifying them in your cluster configuration file.

You can follow these [instructions](../deployments/system-packages.md) to build and push custom Docker images to a registry and configure Cortex to use them.

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```yaml
# docker image paths
image_python_serve: cortexlabs/python-serve:0.15.0
image_python_serve_gpu: cortexlabs/python-serve-gpu:0.15.0
image_tf_serve: cortexlabs/tf-serve:0.15.0
image_tf_serve_gpu: cortexlabs/tf-serve-gpu:0.15.0
image_tf_api: cortexlabs/tf-api:0.15.0
image_onnx_serve: cortexlabs/onnx-serve:0.15.0
image_onnx_serve_gpu: cortexlabs/onnx-serve-gpu:0.15.0
image_operator: cortexlabs/operator:0.15.0
image_manager: cortexlabs/manager:0.15.0
image_downloader: cortexlabs/downloader:0.15.0
image_request_monitor: cortexlabs/request-monitor:0.15.0
image_cluster_autoscaler: cortexlabs/cluster-autoscaler:0.15.0
image_metrics_server: cortexlabs/metrics-server:0.15.0
image_nvidia: cortexlabs/nvidia:0.15.0
image_fluentd: cortexlabs/fluentd:0.15.0
image_statsd: cortexlabs/statsd:0.15.0
image_istio_proxy: cortexlabs/istio-proxy:0.15.0
image_istio_pilot: cortexlabs/istio-pilot:0.15.0
image_istio_citadel: cortexlabs/istio-citadel:0.15.0
image_istio_galley: cortexlabs/istio-galley:0.15.0
```
