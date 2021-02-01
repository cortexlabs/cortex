# Install

## Prerequisites

1. [Docker](https://docs.docker.com/install) must be installed and running on your machine (to verify, check that running `docker ps` does not return an error)
1. Subscribe to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM) (for GPU clusters)
1. An IAM user with `AdministratorAccess` and programmatic access (see [security](security.md) if you'd like to use less privileged credentials after spinning up your cluster)
1. You may need to [request a limit increase](https://console.aws.amazon.com/servicequotas/home?#!/services/ec2/quotas) for your desired instance type

## Spin up Cortex on your AWS account

```bash
# install the CLI
pip install cortex

# spin up Cortex on your AWS account
cortex cluster up  # or: cortex cluster up --config cluster.yaml (see configuration options below)
```

## Configure Cortex

```yaml
# cluster.yaml

# EKS cluster name
cluster_name: cortex

# AWS region
region: us-east-1

# list of availability zones for your region
availability_zones:  # default: 3 random availability zones in your region, e.g. [us-east-1a, us-east-1b, us-east-1c]

# instance type
instance_type: m5.large

# minimum number of instances
min_instances: 1

# maximum number of instances
max_instances: 5

# disk storage size per instance (GB)
instance_volume_size: 50

# instance volume type [gp2 | io1 | st1 | sc1]
instance_volume_type: gp2

# instance volume iops (only applicable to io1)
# instance_volume_iops: 3000

# subnet visibility [public (instances will have public IPs) | private (instances will not have public IPs)]
subnet_visibility: public

# NAT gateway (required when using private subnets) [none | single | highly_available (a NAT gateway per availability zone)]
nat_gateway: none

# API load balancer scheme [internet-facing | internal]
api_load_balancer_scheme: internet-facing

# operator load balancer scheme [internet-facing | internal]
# note: if using "internal", you must configure VPC Peering to connect your CLI to your cluster operator
operator_load_balancer_scheme: internet-facing

# to install Cortex in an existing VPC, you can provide a list of subnets for your cluster to use
# subnet_visibility (specified above in this file) must match your subnets' visibility
# this is an advanced feature (not recommended for first-time users) and requires your VPC to be configured correctly; see https://eksctl.io/usage/vpc-networking/#use-existing-vpc-other-custom-configuration
# here is an example:
# subnets:
#   - availability_zone: us-west-2a
#     subnet_id: subnet-060f3961c876872ae
#   - availability_zone: us-west-2b
#     subnet_id: subnet-0faed05adf6042ab7

# additional tags to assign to AWS resources (all resources will automatically be tagged with cortex.dev/cluster-name: <cluster_name>)
tags:  # <string>: <string> map of key/value pairs

# enable spot instances
spot: false

# SSL certificate ARN (only necessary when using a custom domain)
ssl_certificate_arn:

# primary CIDR block for the cluster's VPC
vpc_cidr: 192.168.0.0/16
```

The docker images used by the Cortex cluster can also be overridden, although this is not common. They can be configured by adding any of these keys to your cluster configuration file (default values are shown):

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```yaml
image_operator: quay.io/cortexlabs/operator:master
image_manager: quay.io/cortexlabs/manager:master
image_downloader: quay.io/cortexlabs/downloader:master
image_request_monitor: quay.io/cortexlabs/request-monitor:master
image_cluster_autoscaler: quay.io/cortexlabs/cluster-autoscaler:master
image_metrics_server: quay.io/cortexlabs/metrics-server:master
image_inferentia: quay.io/cortexlabs/inferentia:master
image_neuron_rtd: quay.io/cortexlabs/neuron-rtd:master
image_nvidia: quay.io/cortexlabs/nvidia:master
image_fluent_bit: quay.io/cortexlabs/fluent-bit:master
image_statsd: quay.io/cortexlabs/statsd:master
image_istio_proxy: quay.io/cortexlabs/istio-proxy:master
image_istio_pilot: quay.io/cortexlabs/istio-pilot:master
```
