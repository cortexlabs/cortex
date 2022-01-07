# Production guide

As you take Cortex from development to production, here are a few pointers that might be useful.

## Use images from a colocated ECR

Configure your cluster and APIs to use images from ECR in the same region as your cluster to accelerate scale-ups, reduce ingress costs, and remove the dependency on Cortex's public quay.io registry.

You can find instructions for mirroring Cortex images [here](../advanced/self-hosted-images.md)

## Handling Cortex updates/upgrades

Use a Route 53 hosted zone as a proxy in front of your Cortex cluster. Every new Cortex cluster provisions a new API load balancer with a unique endpoint. Using a Route 53 hosted zone configured with a subdomain will expose your Cortex cluster API endpoint as a static endpoint (e.g. `cortex.your-company.com`). You will be able to upgrade Cortex versions without downtime, and you will avoid the need to updated your client code every time you migrate to a new cluster. You can find instructions for setting up a custom domain with a Route 53 hosted zone [here](../networking/custom-domain.md), and instructions for updating/upgrading your cluster [here](update.md).

## Production cluster configuration

### Securing your cluster

The following configuration will improve security by preventing your cluster's nodes from being publicly accessible.

```yaml
subnet_visibility: private

nat_gateway: single  # use "highly_available" for large clusters making requests to services outside of the cluster
```

You can make your load balancer private to prevent your APIs from being publicly accessed. In order to access your APIs, you will need to set up VPC peering between the Cortex cluster's VPC and the VPC containing the consumers of the Cortex APIs. See the [VPC peering guide](../networking/vpc-peering.md) for more details.

```yaml
api_load_balancer_scheme: internal
```

You can also restrict access to your load balancers by IP address:

```yaml
api_load_balancer_cidr_white_list: [0.0.0.0/0]
```

These two fields are also available for the operator load balancer. Keep in mind that if you make the operator load balancer private, you'll need to configure VPC peering to use the `cortex` CLI or Python client.

```yaml
operator_load_balancer_scheme: internal
operator_load_balancer_cidr_white_list: [0.0.0.0/0]
```

See [here](../networking/load-balancers.md) for more information about the load balancers.

### Workload load-balancing

Depending on your application's requirements, you might have different needs from the cluster's api load balancer. By default, the api load balancer is a [Network load balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/introduction.html) (NLB). In some situations, a [Classic load balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/introduction.html) (ELB) may be preferred, and can be selected in your cluster config by setting `api_load_balancer_type: elb`. This selection can only be made before creating your cluster.

### Ensure node provisioning

You can take advantage of the cost savings of spot instances and the reliability of on-demand instances by utilizing the `priority` field in node groups. You can deploy two node groups, one that is spot and another that is on-demand. Set the priority of the spot node group to be higher than the priority of the on-demand node group. This encourages the cluster-autoscaler to try to spin up instances from the spot node group first. If there are no more spot instances available, the on-demand node group will be used instead.

```yaml
node_groups:
  - name: gpu-spot
    instance_type: g4dn.xlarge
    min_instances: 0
    max_instances: 5
    spot: true
    priority: 100
  - name: gpu-on-demand
    instance_type: g4dn.xlarge
    min_instances: 0
    max_instances: 5
    priority: 1
```

### Considerations for large clusters

If you plan on scaling your Cortex cluster past 300 nodes or 300 pods, it is recommended to set `prometheus_instance_type` to an instance type with more memory (the default is `t3.medium`, which has 4gb).

## API Spec

### Container design

Configure your health checks to be as accurate as possible to prevent requests from being routed to pods that aren't ready to handle traffic.

### Pods section

Make sure that `max_concurrency` is set to match the concurrency supported by your container.

Tune `max_queue_length` to lower values if you would like to more aggressively redistribute requests to newer pods as your API scales up rather than allowing requests to linger in queues. This would mean that the clients consuming your APIs should implement retry logic with a delay (such as exponential backoff).

### Compute section

Make sure to specify all of the relevant compute resources (especially cpu and memory) to ensure that your pods aren't starved for resources.

### Autoscaling

Revisit the autoscaling docs for [Realtime APIs](../../workloads/realtime/autoscaling.md) and/or [Async APIs](../../workloads/async/autoscaling.md) to effectively handle production traffic by tuning the scaling rate, sensitivity, and over-provisioning.
