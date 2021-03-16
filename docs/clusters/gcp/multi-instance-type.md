# Multi-instance type clusters

The cluster can be configured to provision different instance types depending on what resources the APIs request. The multi instance type cluster has the following advantages over the single-instance type cluster:

* **Lower costs**: Reduced overall compute costs by using the most economical instance for the given workloads.
* **Simpler logistics**: Managing multiple clusters on your own is no longer required.
* **Multi-purpose cluster**: The cluster can now take any range of workloads. One cluster for everything. Just throw a bunch of node pools in the cluster config, and you’re set.

## Best practices

When specifying the node pools in your `cluster.yaml` config, keep in mind that node pools with lower indexes have a higher priority over the other ones. With that mind, the best practices that result from this are:

1. Node pools with smaller instances should have the higher priority.
1. Node pools with CPU-only instances should come before the node pools equipped with GPU instances.
1. The preemptible node pools should always come first over the ones that have on-demand instances.

## Example node pools

### CPU preemptible/on-demand with GPU on-demand

```yaml
# cluster.yaml

node_pools:
  - name: cpu-preempt
    instance_type: e2-standard-2
    preemptible: true
  - name: cpu
    instance_type: e2-standard-2
  - name: gpu
    instance_type: e2-standard-2
    accelerator_type: nvidia-tesla-t4
```

### CPU on-demand with 2 GPU on-demand

```yaml
# cluster.yaml

node_pools:
  - name: cpu
    instance_type: e2-standard-2
  - name: gpu-small
    instance_type: e2-standard-2
    accelerator_type: nvidia-tesla-t4
  - name: gpu-large
    instance_type: e2-standard-2
    accelerator_type: nvidia-tesla-t4
    accelerators_per_instance: 4
```

### 3 preemptible CPU node pools with 1 on-demand CPU

```yaml
# cluster.yaml

node_pools:
  - name: cpu-0
    instance_type: e2-standard-2
    preemptible: true
  - name: cpu-1
    instance_type: e2-standard-4
    preemptible: true
  - name: cpu-2
    instance_type: e2-standard-8
    preemptible: true
  - name: cpu-3
    instance_type: e2-standard-32
```
