# Cluster configuration

The Cortex cluster may be configured by providing a configuration file to `cortex cluster install` or `cortex cluster update` via the  `--config` flag (e.g. `cortex cluster install --config=cluster.yaml`). Below is the schema for the cluster configuration file, with default values shown:

<!-- CORTEX_VERSION_BRANCH_STABLE -->

```yaml
# cluster.yaml

# AWS credentials (if not specified, ~/.aws/credentials will be checked) (can be overriden by $AWS_ACCESS_KEY_ID and $AWS_SECRET_ACCESS_KEY)
aws_access_key_id: ***
aws_secret_access_key: ***

# Optional AWS credentials for the Operator which may be used to restrict its AWS access (defaults to the AWS credentials set above)
cortex_aws_access_key_id: ***
cortex_aws_secret_access_key: ***

# Instance type Cortex will use
instance_type: m5.large

# Minimum and maximum number of instances in the cluster
min_instances: 2
max_instances: 5

# Name of the S3 bucket Cortex will use
bucket: cortex-[RANDOM_ID]

# Region Cortex will use
region: us-west-2

# Name of the CloudWatch log group Cortex will use
log_group: cortex

# Name of the EKS cluster Cortex will create
cluster_name: cortex

# Flag to enable collection of anonymous usage stats and error reports
telemetry: true

# Image paths
image_tf_serve: cortexlabs/tf-serve:master
image_tf_serve_gpu: cortexlabs/tf-serve-gpu:master
image_onnx_serve: cortexlabs/onnx-serve:master
image_onnx_serve_gpu: cortexlabs/onnx-serve-gpu:master
image_operator: cortexlabs/operator:master
image_manager: cortexlabs/manager:master
image_tf_api: cortexlabs/tf-api:master
image_downloader: cortexlabs/downloader:master
image_cluster_autoscaler: cortexlabs/cluster-autoscaler:master
image_metrics_server: cortexlabs/metrics-server:master
image_nvidia: cortexlabs/nvidia:master
image_fluentd: cortexlabs/fluentd:master
image_statsd: cortexlabs/statsd:master
image_istio_proxy: cortexlabs/istio-proxy:master
image_istio_pilot: cortexlabs/istio-pilot:master
image_istio_citadel: cortexlabs/istio-citadel:master
image_istio_galley: cortexlabs/istio-galley:master
```
