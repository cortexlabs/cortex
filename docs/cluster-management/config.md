# Cluster configuration

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

The Cortex cluster may be configured by providing a configuration file to `cortex cluster up` or `cortex cluster configure` via the `--config` flag (e.g. `cortex cluster up --config cluster.yaml`). Below is the schema for the cluster configuration file, with default values shown (unless otherwise specified):

<!-- CORTEX_VERSION_MINOR x6 -->
```yaml
# cluster.yaml

# EKS cluster name for cortex (default: cortex)
cluster_name: cortex

# AWS region
region: us-east-1

# S3 bucket (default: <cluster_name>-<RANDOM_ID>)
# note: your cortex cluster uses this bucket for metadata storage, and it should not be accessed directly (a separate bucket should be used for your models)
bucket: # cortex-<RANDOM_ID>

# list of availability zones for your region (default: 3 random availability zones from the specified region)
availability_zones: # e.g. [us-east-1a, us-east-1b, us-east-1c]

# instance type
instance_type: m5.large

# minimum number of instances (must be >= 0)
min_instances: 1

# maximum number of instances (must be >= 1)
max_instances: 5

# disk storage size per instance (GB) (default: 50)
instance_volume_size: 50

# instance volume type [gp2, io1, st1, sc1] (default: gp2)
instance_volume_type: gp2

# instance volume iops (only applicable to io1 storage type) (default: 3000)
# instance_volume_iops: 3000

# whether the subnets used for EC2 instances should be public or private (default: "public")
# if "public", instances will be assigned public IP addresses; if "private", instances won't have public IPs and a NAT gateway will be created to allow outgoing network requests
# see https://docs.cortex.dev/v/master/miscellaneous/security#private-cluster for more information
subnet_visibility: public  # must be "public" or "private"

# whether to include a NAT gateway with the cluster (a NAT gateway is necessary when using private subnets)
# default value is "none" if subnet_visibility is set to "public"; "single" if subnet_visibility is "private"
nat_gateway: none  # must be "none", "single", or "highly_available" (highly_available means one NAT gateway per availability zone)

# whether the API load balancer should be internet-facing or internal (default: "internet-facing")
# note: if using "internal", APIs will still be accessible via the public API Gateway endpoint unless you also disable API Gateway in your API's configuration (if you do that, you must configure VPC Peering to connect to your APIs)
# see https://docs.cortex.dev/v/master/miscellaneous/security#private-cluster for more information
api_load_balancer_scheme: internet-facing  # must be "internet-facing" or "internal"

# whether the operator load balancer should be internet-facing or internal (default: "internet-facing")
# note: if using "internal", you must configure VPC Peering to connect your CLI to your cluster operator (https://docs.cortex.dev/v/master/guides/vpc-peering)
# see https://docs.cortex.dev/v/master/miscellaneous/security#private-operator for more information
operator_load_balancer_scheme: internet-facing  # must be "internet-facing" or "internal"

# whether to disable API gateway cluster-wide
# if set to "public" (the default), each API can specify whether to use API Gateway
# if set to "none", no APIs will be allowed to use API Gateway
api_gateway: public  # must be "public" or "none"

# additional tags to assign to aws resources for labelling and cost allocation (by default, all resources will be tagged with cortex.dev/cluster-name=<cluster_name>)
tags:  # <string>: <string> map of key/value pairs

# whether to use spot instances in the cluster (default: false)
# see https://docs.cortex.dev/v/master/cluster-management/spot-instances for additional details on spot configuration
spot: false

# see https://docs.cortex.dev/v/master/guides/custom-domain for instructions on how to set up a custom domain
ssl_certificate_arn:

# primary CIDR block for the cluster's VPC (default: 192.168.0.0/16)
# vpc_cidr: 192.168.0.0/16
```

The default docker images used for your Predictors are listed in the instructions for [system packages](../deployments/system-packages.md), and can be overridden in your [Realtime API configuration](../deployments/realtime-api/api-configuration.md) and in your [Batch API configuration](../deployments/batch-api/api-configuration.md).

The docker images used by the Cortex cluster can also be overridden, although this is not common. They can be configured by adding any of these keys to your cluster configuration file (default values are shown):

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```yaml
# docker image paths
image_operator: quay.io/cortexlabs/operator:master
image_manager: quay.io/cortexlabs/manager:master
image_downloader: quay.io/cortexlabs/downloader:master
image_request_monitor: quay.io/cortexlabs/request-monitor:master
image_cluster_autoscaler: quay.io/cortexlabs/cluster-autoscaler:master
image_metrics_server: quay.io/cortexlabs/metrics-server:master
image_inferentia: quay.io/cortexlabs/inferentia:master
image_neuron_rtd: quay.io/cortexlabs/neuron-rtd:master
image_nvidia: quay.io/cortexlabs/nvidia:master
image_fluentd: quay.io/cortexlabs/fluentd:master
image_statsd: quay.io/cortexlabs/statsd:master
image_istio_proxy: quay.io/cortexlabs/istio-proxy:master
image_istio_pilot: quay.io/cortexlabs/istio-pilot:master
```
