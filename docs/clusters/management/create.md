# Install

## Prerequisites

1. Install and run [Docker](https://docs.docker.com/install) on your machine.
1. Subscribe to the [AMI with GPU support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM) (for GPU clusters).
1. Create an IAM user with `AdministratorAccess` and programmatic access.
1. You may need to [request limit increases](https://console.aws.amazon.com/servicequotas/home?#!/services/ec2/quotas) for your desired instance types.

## Create a cluster on your AWS account

```bash
# install the CLI
pip install cortex

# create a cluster
cortex cluster up cluster.yaml
```

## `cluster.yaml`

```yaml
# cluster name
cluster_name: cortex

# AWS region
region: us-east-1

# list of availability zones for your region
availability_zones:  # default: 3 random availability zones in your region, e.g. [us-east-1a, us-east-1b, us-east-1c]

# list of cluster node groups; the smaller index, the higher the priority of the node group
node_groups:
  - name: ng-cpu # name of the node group
    instance_type: m5.large # instance type
    min_instances: 1 # minimum number of instances
    max_instances: 5 # maximum number of instances
    instance_volume_size: 50 # disk storage size per instance (GB)
    instance_volume_type: gp2 # instance volume type [gp2 | io1 | st1 | sc1]
    # instance_volume_iops: 3000 # instance volume iops (only applicable to io1)
    spot: false # enable spot instances

  - name: ng-gpu
    instance_type: g4dn.xlarge
    min_instances: 1
    max_instances: 5
    instance_volume_size: 50
    instance_volume_type: gp2
    # instance_volume_iops: 3000
    spot: false

  - name: ng-inferentia
    instance_type: inf1.xlarge
    min_instances: 1
    max_instances: 5
    instance_volume_size: 50
    instance_volume_type: gp2
    # instance_volume_iops: 3000
    spot: false
  ...

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

# SSL certificate ARN (only necessary when using a custom domain)
ssl_certificate_arn:

# List of IAM policies to attach to your Cortex APIs
iam_policy_arns: ["arn:aws:iam::aws:policy/AmazonS3FullAccess"]

# primary CIDR block for the cluster's VPC
vpc_cidr: 192.168.0.0/16
```

The docker images used by the cluster can also be overridden. They can be configured by adding any of these keys to your cluster configuration file (default values are shown):

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```yaml
image_operator: quay.io/cortexlabs/operator:master
image_manager: quay.io/cortexlabs/manager:master
image_downloader: quay.io/cortexlabs/downloader:master
image_request_monitor: quay.io/cortexlabs/request-monitor:master
image_image_async_gateway: quay.io/cortexlabs/async-gateway:master
image_cluster_autoscaler: quay.io/cortexlabs/cluster-autoscaler:master
image_metrics_server: quay.io/cortexlabs/metrics-server:master
image_inferentia: quay.io/cortexlabs/inferentia:master
image_neuron_rtd: quay.io/cortexlabs/neuron-rtd:master
image_nvidia: quay.io/cortexlabs/nvidia:master
image_fluent_bit: quay.io/cortexlabs/fluent-bit:master
image_istio_proxy: quay.io/cortexlabs/istio-proxy:master
image_istio_pilot: quay.io/cortexlabs/istio-pilot:master
image_prometheus: quay.io/cortexlabs/prometheus:master
image_prometheus_config_reloader: quay.io/cortexlabs/prometheus-config-reloader:master
image_prometheus_operator: quay.io/cortexlabs/prometheus-operator:master
image_prometheus_statsd_exporter: quay.io/cortexlabs/prometheus-statsd-exporter:master
image_prometheus_dcgm_exporter: quay.io/cortexlabs/prometheus-dcgm-exporter:master
image_prometheus_kube_state_metrics: quay.io/cortexlabs/prometheus-kube-state-metrics:master
image_prometheus_node_exporter: quay.io/cortexlabs/prometheus-node-exporter:master
image_kube_rbac_proxy: quay.io/cortexlabs/kube-rbac-proxy:master
image_grafana: quay.io/cortexlabs/grafana:master
image_event_exporter: quay.io/cortexlabs/event-exporter:master
```
