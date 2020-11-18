# Install

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Spin up Cortex on your AWS account

Prerequisite: [Docker](https://docs.docker.com/install).

```bash
# install the CLI
pip install cortex

# spin up Cortex on your AWS account
cortex cluster up # || cortex cluster up --config cluster.yaml (see configuration options below)

# set the default environment
cortex env default aws
```

## Configure your cluster

<!-- CORTEX_VERSION_MINOR x6 -->
<!-- CORTEX_VERSION_BRANCH_STABLE -->

```yaml
# cluster.yaml

# EKS cluster name
cluster_name: cortex

# AWS region
region: us-east-1

# S3 bucket for metadata storage, it should not be used for storing models
bucket: # <cluster_name>-<RANDOM_ID>

# list of availability zones for your region
availability_zones: [us-east-1a, us-east-1b, us-east-1c]

# instance type
instance_type: m5.large

# minimum number of instances (must be >= 0)
min_instances: 1

# maximum number of instances (must be >= 1)
max_instances: 5

# disk storage size per instance (GB)
instance_volume_size: 50

# instance volume type [gp2|io1|st1|sc1]
instance_volume_type: gp2

# instance volume iops (only applicable to io1)
# instance_volume_iops: 3000

# subnet visibility [public (instances will have public IPs) | private (instances will not have public IPs)]
subnet_visibility: public

# NAT gateway (required when using private subnets) [none | single | highly_available (a NAT gateway per availability zone)]
nat_gateway: none

# API load balancer scheme [internet-facing|internal]
api_load_balancer_scheme: internet-facing

# operator load balancer scheme [internet-facing|internal]
operator_load_balancer_scheme: internet-facing

# API gateway [public|none]
api_gateway: public

# tags [<string>:<string>]
tags: cortex.dev/cluster-name=<cluster_name>

# enable spot instances
spot: false

# SSL certificate ARN
ssl_certificate_arn:

# primary CIDR block for the cluster's VPC
vpc_cidr: 192.168.0.0/16

# docker images
image_operator: cortexlabs/operator:master
image_manager: cortexlabs/manager:master
image_downloader: cortexlabs/downloader:master
image_request_monitor: cortexlabs/request-monitor:master
image_cluster_autoscaler: cortexlabs/cluster-autoscaler:master
image_metrics_server: cortexlabs/metrics-server:master
image_inferentia: cortexlabs/inferentia:master
image_neuron_rtd: cortexlabs/neuron-rtd:master
image_nvidia: cortexlabs/nvidia:master
image_fluentd: cortexlabs/fluentd:master
image_statsd: cortexlabs/statsd:master
image_istio_proxy: cortexlabs/istio-proxy:master
image_istio_pilot: cortexlabs/istio-pilot:master
```

## Advanced

* [Security](security.md).
* [VPC peering](vpc-peering.md).
* [Custom domain](custom-domain.md).
* [REST API Gateway](rest-api-gateway.md).
* [Spot instances](spot.md).
* [SSH into instances](ssh.md).

## Troubleshooting

* See [EKS-optimized AMI with GPU support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM) for GPU instance issues.
* See [EC2 service quotas](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-resource-limits.html) for instance limit issues.
