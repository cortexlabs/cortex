# Multi-instance type clusters

Cortex can be configured to provision different instance types to improve workload performance and reduce cloud infrastructure spend.

## Best practices

**Node groups with lower indices have higher priority.**

1. Small instance node groups should be listed before large instance node groups.
1. CPU node groups should be listed before GPU/Inferentia node groups.
1. Spot node groups should always be listed before on-demand node groups.

## Examples

### CPU spot, CPU on-demand, and GPU on-demand

```yaml
# cluster.yaml

node_groups:
  - name: cpu-spot
    instance_type: m5.large
    spot: true
  - name: cpu-on-demand
    instance_type: m5.large
  - name: gpu-on-demand
    instance_type: g4dn.xlarge
```

### CPU on-demand, GPU on-demand, and Inferentia on-demand

```yaml
# cluster.yaml

node_groups:
  - name: cpu-on-demand
    instance_type: m5.large
  - name: gpu-on-demand
    instance_type: g4dn.xlarge
  - name: inferentia-on-demand
    instance_type: inf.xlarge
```

### 3 CPU spot and 1 CPU on-demand

```yaml
# cluster.yaml

node_groups:
  - name: cpu-1
    instance_type: t3.medium
    spot: true
  - name: cpu-2
    instance_type: m5.2xlarge
    spot: true
  - name: cpu-3
    instance_type: m5.8xlarge
    spot: true
  - name: cpu-4
    instance_type: m5.24xlarge
```
