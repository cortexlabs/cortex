# Multi-instance type clusters

The cluster can be configured to provision different instance types depending on what resources the APIs request. The multi instance type cluster has the following advantages over the single-instance type cluster:

* **Lower costs**: Reduced overall compute costs by using the most economical instance for the given workloads.
* **Simpler logistics**: Managing multiple clusters on your own is no longer required.
* **Multi-purpose cluster**: The cluster can now take any range of workloads. One cluster for everything. Just throw a bunch of node groups in the cluster config, and you’re set.

## Best practices

When specifying the node groups in your `cluster.yaml` config, keep in mind that node groups with lower indexes have a higher priority over the other ones. With that mind, the best practices that result from this are:

1. Node groups with smaller instances should have the higher priority.
1. Node groups with CPU-only instances should come before the node groups equipped with GPU/Inferentia instances.
1. The spot node groups should always come first over the ones that have on-demand instances.

## Example node groups

### CPU spot/on-demand with GPU on-demand

```yaml
# cluster.yaml

node_groups:
  - name: cpu-spot
    instance_type: m5.large
    spot: true
  - name: cpu
    instance_type: m5.large
  - name: gpu
    instance_type: g4dn.xlarge
```

### CPU on-demand, GPU on-demand and Inferentia on-demand

```yaml
# cluster.yaml

node_groups:
  - name: cpu
    instance_type: m5.large
  - name: gpu
    instance_type: g4dn.xlarge
  - name: inferentia
    instance_type: inf.xlarge
```

### 3 spot CPU node groups with 1 on-demand CPU

```yaml
# cluster.yaml

node_groups:
  - name: cpu-0
    instance_type: t3.medium
    spot: true
  - name: cpu-1
    instance_type: m5.2xlarge
    spot: true
  - name: cpu-2
    instance_type: m5.8xlarge
    spot: true
  - name: cpu-3
    instance_type: m5.24xlarge
```

The above can also be achieved with the following config.

```yaml
# cluster.yaml

node_groups:
  - name: cpu-0
    instance_type: t3.medium
    spot: true
    spot_config:
      instance_distribution: [m5.2xlarge, m5.8xlarge]
      max_price: 3.27
  - name: cpu-1
    instance_type: m5.24xlarge
```
