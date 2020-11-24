# Install

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Spin up Cortex on your AWS account

Make sure [Docker](https://docs.docker.com/install) is running on your machine.

If you're using GPUs, subscribe to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM) before creating your cluster.

```bash
# install the CLI
pip install cortex

# spin up Cortex on your AWS account
cortex cluster up  # or: cortex cluster up --config cluster.yaml (see configuration options below)

# set the default environment
cortex env default aws
```

<!-- CORTEX_VERSION_MINOR -->
Try the [tutorial](../../examples/pytorch/text-generator/README.md) or deploy one of our [examples](https://github.com/cortexlabs/cortex/tree/master/examples).

## Configure Cortex

<!-- CORTEX_VERSION_MINOR -->
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
# note: if using "internal", you must configure VPC Peering to connect your CLI to your cluster operator (https://docs.cortex.dev/v/master/aws/vpc-peering)
operator_load_balancer_scheme: internet-facing

# API Gateway [public (API Gateway will be used by default, can be disabled per API) | none (API Gateway will be disabled for all APIs)]
api_gateway: public

# additional tags to assign to AWS resources (all resources will automatically be tagged with cortex.dev/cluster-name: <cluster_name>)
tags:  # <string>: <string> map of key/value pairs

# enable spot instances
spot: false

# SSL certificate ARN (only necessary when using a custom domain without API Gateway)
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
image_fluentd: quay.io/cortexlabs/fluentd:master
image_statsd: quay.io/cortexlabs/statsd:master
image_istio_proxy: quay.io/cortexlabs/istio-proxy:master
image_istio_pilot: quay.io/cortexlabs/istio-pilot:master
```

The default docker images used for your Predictors are listed in the instructions for [system packages](../deployments/system-packages.md), and can be overridden in your [Realtime API configuration](../deployments/realtime-api/api-configuration.md) and in your [Batch API configuration](../deployments/batch-api/api-configuration.md).

## Advanced

* [Security](security.md)
* [VPC peering](vpc-peering.md)
* [Custom domain](custom-domain.md)
* [REST API Gateway](rest-api-gateway.md)
* [Spot instances](spot.md)
* [SSH into instances](ssh.md)
