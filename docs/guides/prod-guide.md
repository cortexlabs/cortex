# Production guide

As you try to take Cortex from development to production, here are a few recommendations and pointers that might be useful.

## Use images from a colocated ECR

Configure your cluster configuration and API spec to use images from ECR in the same region as your cluster to accelerate scale-ups, reduce ingress costs and remove dependency on Cortex's public quay.io registry.

You can find instructions for mirroring Cortex images [here](./self-hosted-images.md)

## Handling Cortex updates/upgrades

Use a route 53 hosted zone as a proxy in front of your Cortex cluster. Every new Cortex cluster provisions a new API load balancer with a unique endpoint. Using a route 53 hosted zone configured with a subdomain will expose your Cortex cluster API endpoint as a single endpoint e.g. `cortex.your-company.com`. You will be able to upgrade Cortex versions with minimal downtime and avoid needing to change your client code every time you migrate to a new cluster. You can find instructions for setting up a custom domain with route 53 hosted zone [here](./custom-domain.md).

## Production cluster configuration

### Securing your cluster

The following configuration will improve security by preventing your cluster's nodes from being publicly accessible.

```yaml
subnet_visibility: private

nat_gateway: single # for large clusters making requests to services outside the cluster (e.g. S3 or database) use highly_available
```

You can make your load balancers private to prevent your APIs from being publicly accessed. In order to access your APIs, you will need to set up VPC peering between the Cortex cluster's VPC and the VPC containing the consumers of the Cortex APIs. See the [VPC peering guide](./vpc-peering.md) for more details.

```yaml
api_load_balancer_scheme: internal
```

You can also restrict access to your load balancers by IP address to improve security even further. This can be done if the API load balancer is public or private.

```yaml
api_load_balancer_cidr_white_list: [0.0.0.0/0]
```

### Ensure node provisioning

You can get the cost savings of spot instances and the reliability of on-demand instances by taking advantage of the priority field in node groups. You can deploy two node groups, one that is spot and another that is on-demand. Set the priority of the spot node group to be higher than the priority of the on-demand node group. This encourages the cluster-autoscaler to try to spin up instances from the spot node group first. If after 5 minutes, the spot node group is unable to scale up because there are no more spot instances available, the on-demand node group  will be used instead.

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

If you plan on spinning up a Cortex cluster reaching 400 nodes or 1000 pods, you might want to consider setting `prometheus_instance_type` to a larger instance type. A good rule of thumb is a t3.medium instance can reliably handle 400 nodes and 800 pods.

## API Spec

### Container design

Configure your health checks to be as accurate as possible to mitigate the chances of requests being routed to pods that are not ready yet.

### Pods section

Make sure that `max_concurrency` is set to match the concurrency supported by your container.

Tune `max_queue_length` to lower values if you would like to more aggressively redistribute requests to newer pods as your API scales up rather than allowing requests to linger in queues. This would mean that the clients consuming your APIs should implement a retry logic with a delay (such as exponential backoff).

### Compute section

Make sure to specify all of the relevant compute resources, especially the cpu and memory to ensure that your pods aren't starved for resources.

### Autoscaling

Revisit the autoscaling docs for your [Realtime APIs](../workloads/realtime/autoscaling.md) and [Async APIs](../workloads/async/autoscaling.md) to effectively handle production traffic by tuning the sensitivity, scaling rate and over-provisioning.
