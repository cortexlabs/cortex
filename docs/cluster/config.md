# Cluster configuration

The Cortex cluster may be configured by providing a configuration file to `cortex cluster up` or `cortex cluster update` via the  `--config` flag (e.g. `cortex cluster up --config=cluster.yaml`). Below is the schema for the cluster configuration file, with default values shown:

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

# Volume size for each instance in the cluster (in GiB)
instance_volume_size: 50

# Flag to enable collection of anonymous usage stats and error reports
telemetry: true

# Image paths
image_predictor_serve: cortexlabs/predictor-serve:0.10.2
image_predictor_serve_gpu: cortexlabs/predictor-serve-gpu:0.10.2
image_tf_serve: cortexlabs/tf-serve:0.10.2
image_tf_serve_gpu: cortexlabs/tf-serve-gpu:0.10.2
image_tf_api: cortexlabs/tf-api:0.10.2
image_onnx_serve: cortexlabs/onnx-serve:0.10.2
image_onnx_serve_gpu: cortexlabs/onnx-serve-gpu:0.10.2
image_operator: cortexlabs/operator:0.10.2
image_manager: cortexlabs/manager:0.10.2
image_downloader: cortexlabs/downloader:0.10.2
image_cluster_autoscaler: cortexlabs/cluster-autoscaler:0.10.2
image_metrics_server: cortexlabs/metrics-server:0.10.2
image_nvidia: cortexlabs/nvidia:0.10.2
image_fluentd: cortexlabs/fluentd:0.10.2
image_statsd: cortexlabs/statsd:0.10.2
image_istio_proxy: cortexlabs/istio-proxy:0.10.2
image_istio_pilot: cortexlabs/istio-pilot:0.10.2
image_istio_citadel: cortexlabs/istio-citadel:0.10.2
image_istio_galley: cortexlabs/istio-galley:0.10.2
```
