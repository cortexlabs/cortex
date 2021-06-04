# Overview

## Cluster

The Cortex cluster is an EKS (managed kubernetes) cluster that lives in a dedicated VPC on your AWS account.

### Worker node groups

The kubernetes cluster uses EC2 autoscaling groups for its worker node groups. It can take advantage of reserved/spot instances to help reduce costs.

Cortex supports most EC2 instance types. The necessary device drivers are installed to expose GPUs and inferentia chips to your workloads.

Cortex uses the Kubernetes Cluster Autoscaler to scale the appropriate node groups and satisfy the compute demands of your workloads.

### Networking

By default, a new dedicated VPC is created for the cluster during installation.

Two NLB loadbalancers are set up to route traffic to the cluster. One loadbalancer is dedicated for traffic to your APIs and the other loadbalancer is dedicated for requests to Cortex from your CLI or your python client. Traffic to the loadbalancers can be secured and restricted based on your cluster configuration.

### Observability

All logs from the Cortex cluster are pushed to a CloudWatch log group using FluentBit. An in-cluster prometheus installation is used to scrape and collect metrics for observability and autoscaling purposes. Metrics and dashboards pertaining to your APIs and instance usage can be viewed and explored on Grafana.

## Deploying to the cluster

After a successful Cortex cluster installation, you can use the Cortex CLI or Python Client to deploy different API kinds. The clients use AWS credentials to authenticate to the Cortex cluster.

Cortex uses a collection of containers, referred to as pod, as the atomic unit. The scaling and replication occurs at the pod level. The orchestration and scaling of pods is unique to the different API kinds:

* Realtime
* Async
* Batch
* Task

Visit the API specific documentation for more details.

## Cluster Diagram

![]()