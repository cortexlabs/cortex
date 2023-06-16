# Install

## Prerequisites

1. Install and run [Docker](https://docs.docker.com/install) on your machine.
1. Subscribe to the [AMI with GPU support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM) (for GPU clusters).
1. Create an IAM user with `AdministratorAccess` and programmatic access.
1. You may need to [request limit increases](https://console.aws.amazon.com/servicequotas/home?#!/services/ec2/quotas) for your desired instance types.

## Create a cluster on your AWS account

<!-- CORTEX_VERSION_README -->
```bash
# install the cortex CLI
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.42.2/get-cli.sh)"

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

# list of cluster node groups;
node_groups:
  - name: ng-cpu # name of the node group
    instance_type: m5.large # instance type
    min_instances: 1 # minimum number of instances
    max_instances: 5 # maximum number of instances
    priority: 1 # priority of the node group; the higher the value, the higher the priority [1-100]
    instance_volume_size: 50 # disk storage size per instance (GB)
    instance_volume_type: gp3 # instance volume type [gp2 | gp3 | io1 | st1 | sc1]
    # instance_volume_iops: 3000 # instance volume iops (only applicable to io1/gp3)
    # instance_volume_throughput: 125 # instance volume throughput (only applicable to gp3)
    spot: false # whether to use spot instances

  - name: ng-gpu
    instance_type: g4dn.xlarge
    min_instances: 1
    max_instances: 5
    instance_volume_size: 50
    instance_volume_type: gp3
    spot: false
  # ...

# subnet visibility for instances [public (instances will have public IPs) | private (instances will not have public IPs)]
# when using private subnets, you may wish to enable VPC endpoints (via the AWS console) for S3 and ECR to avoid extra NAT Gateway charges
subnet_visibility: public

# NAT gateway (required when using private subnets) [none | single | highly_available (a NAT gateway per availability zone)]
nat_gateway: none

# API load balancer type [nlb | elb]
api_load_balancer_type: nlb

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

# restrict access to APIs by cidr blocks/ip address ranges
api_load_balancer_cidr_white_list: [0.0.0.0/0]

# restrict access to the Operator by cidr blocks/ip address ranges
operator_load_balancer_cidr_white_list: [0.0.0.0/0]

# additional tags to assign to AWS resources (all resources will automatically be tagged with cortex.dev/cluster-name: <cluster_name>)
tags:  # <string>: <string> map of key/value pairs

# SSL certificate ARN (only necessary when using a custom domain)
ssl_certificate_arn:

# list of IAM policies to attach to your Cortex APIs
iam_policy_arns: ["arn:aws:iam::aws:policy/AmazonS3FullAccess"]

# primary CIDR block for the cluster's VPC
vpc_cidr: 192.168.0.0/16

# instance type for prometheus (use an instance with more memory for clusters exceeding 300 nodes or 300 pods)
prometheus_instance_type: "t3.medium"
```

The docker images used by the cluster can also be overridden. They can be configured by adding any of these keys to your cluster configuration file (default values are shown):

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```yaml
image_manager: quay.io/cortexlabs/manager:master
image_operator: quay.io/cortexlabs/operator:master
image_controller_manager: quay.io/cortexlabs/controller-manager:master
image_autoscaler: quay.io/cortexlabs/autoscaler:master
image_proxy: quay.io/cortexlabs/proxy:master
image_async_gateway: quay.io/cortexlabs/async-gateway:master
image_activator: quay.io/cortexlabs/activator:master
image_enqueuer: quay.io/cortexlabs/enqueuer:master
image_dequeuer: quay.io/cortexlabs/dequeuer:master
image_cluster_autoscaler: quay.io/cortexlabs/cluster-autoscaler:master
image_metrics_server: quay.io/cortexlabs/metrics-server:master
image_nvidia_device_plugin: quay.io/cortexlabs/nvidia-device-plugin:master
image_neuron_device_plugin: quay.io/cortexlabs/neuron-device-plugin:master
image_neuron_scheduler: quay.io/cortexlabs/neuron-scheduler:master
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
image_kubexit: quay.io/cortexlabs/kubexit:master
```
