# Multi-instance

Cortex can be configured to provision different instance types to improve workload performance and reduce cloud infrastructure spend.

## Best practices

**Node groups with lower indices have higher priority.**

1. Spot node groups should be listed before on-demand node groups.
2. CPU node groups should be listed before GPU/Inferentia node groups.
3. Node groups with small instance types should be listed before node groups with large instance types.

## Examples

### CPU spot cluster, with on-demand backup

```yaml
# cluster.yaml

node_groups:
  - name: cpu-spot
    instance_type: m5.large
    spot: true
    spot_config:
      instance_distribution: [m5a.large, m5d.large, m5n.large, m5ad.large, m5dn.large, m4.large, t3.large, t3a.large, t2.large]
  - name: cpu-on-demand
    instance_type: m5.large
```

### On-demand cluster supporting CPU, GPU, and Inferentia

```yaml
# cluster.yaml

node_groups:
  - name: cpu
    instance_type: m5.large
  - name: gpu
    instance_type: g4dn.xlarge
  - name: inf
    instance_type: inf.xlarge
```

### Spot cluster supporting CPU and GPU \(with on-demand backup\)

```yaml
# cluster.yaml

node_groups:
  - name: cpu-spot
    instance_type: m5.large
    spot: true
    spot_config:
      instance_distribution: [m5a.large, m5d.large, m5n.large, m5ad.large, m5dn.large, m4.large, t3.large, t3a.large, t2.large]
  - name: cpu-on-demand
    instance_type: m5.large
  - name: gpu-spot
    instance_type: g4dn.xlarge
    spot: true
  - name: gpu-on-demand
    instance_type: g4dn.xlarge
```

### CPU spot cluster with multiple instance types and on-demand backup

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

