# Cluster configuration

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

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

# whether the subnets used for EC2 instances should be public or private (default: "public")
subnet_visibility: public  # must be "public" or "private"

# whether to include a NAT gateway with the cluster (a NAT gateway is necessary when using private subnets)
# default value is "none" if subnet_visibility is set to "public"; "single" if subnet_visibility is set to "private"
nat_gateway: none  # must be "none", "single", and "highly_available" (one NAT gateway per availability zone)

# whether the API load balancer should be internet-facing or internal (default: "internet-facing")
# note: if using "internal", you must configure VPC Peering or an API Gateway VPC Link to connect to your APIs (see www.cortex.dev/guides/vpc-peering or www.cortex.dev/guides/api-gateway)
api_load_balancer_scheme: internet-facing  # must be "internet-facing" or "internal"

# whether the Operator load balancer should be internet-facing or internal (default: "internet-facing")
# note: if using "internal", you must configure VPC Peering to connect your CLI to your cluster operator (see www.cortex.dev/guides/vpc-peering)
operator_load_balancer_scheme: internet-facing  # must be "internet-facing" or "internal"

# CloudWatch log group for cortex (default: <cluster_name>)
log_group: cortex

# whether to use spot instances in the cluster (default: false)
# see https://cortex.dev/v/master/cluster-management/spot-instances for additional details on spot configuration
spot: false
```

The default docker images used for your Predictors are listed in the instructions for [system packages](../deployments/system-packages.md), and can be overridden in your [API configuration](../deployments/api-configuration.md).

The docker images used by the Cortex cluster can also be overriden, although this is not common. They can be configured by adding any of these keys to your cluster configuration file (default values are shown):

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```yaml
# docker image paths
image_operator: cortexlabs/operator:master
image_manager: cortexlabs/manager:master
image_downloader: cortexlabs/downloader:master
image_request_monitor: cortexlabs/request-monitor:master
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
