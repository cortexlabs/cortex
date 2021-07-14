# Spot instances

```yaml
# cluster.yaml

node_groups:
  - name: node-group-1

    # whether to use spot instances for this node group (default: false)
    spot: false  # this must be set to true to use spot instances

    spot_config:
      # additional instance types with identical or better specs than the primary cluster instance type (defaults to only the primary instance type)
      instance_distribution: # [similar_instance_type_1, similar_instance_type_2]

      # minimum number of on demand instances (default: 0)
      on_demand_base_capacity: 0

      # percentage of on demand instances to use after the on demand base capacity has been met [0, 100] (default: 50)
      # note: setting this to 0 may hinder cluster scale-up when spot instances are not available
      on_demand_percentage_above_base_capacity: 0

      # max price for spot instances (default: the on-demand price of the primary instance type)
      max_price: # <float>

      # number of spot instance pools across which to allocate spot instances [1, 20] (default: number of instances in instance distribution)
      instance_pools: 3
```

Spot instances are not guaranteed to be available. The chances of getting spot instances can be improved by providing `instance_distribution`, a list of alternative instance types to the primary `instance_type` you specified. If left blank, Cortex will only include the primary instance type in the `instance_distribution`. When using `instance_distribution`, use the instance type with the fewest compute resources as your primary `instance_type`. Note that the default value for `max_price` is the on-demand price of the primary instance type, but you may wish to set this to the on-demand price of the most expensive instance type in your `instance_distribution`.

Spot instances can be mixed with on-demand instances in a single node group by configuring `on_demand_base_capacity` and `on_demand_percentage_above_base_capacity`. `on_demand_base_capacity` enforces the minimum number of nodes that will be fulfilled by on-demand instances as your cluster is scaling up. `on_demand_percentage_above_base_capacity` defines the percentage of instances that will be on-demand after the base capacity has been fulfilled (the rest being spot instances). `instance_pools` is the number of pools per availability zone to allocate your instances from. See [here](https://docs.aws.amazon.com/autoscaling/ec2/APIReference/API_InstancesDistribution.html) for more details.

Even if multiple instances are specified in your `instance_distribution`, it is still possible that AWS will not be able to provision a spot instance when requested. One possibility is that AWS has exhausted all of the available spot instances of your requested type(s) in your availability zones. Another possibility is that the current price of your requested instance type(s) is higher than your `max_price`. To mitigate this, you may add a second node group to your cluster configuration which is configured to use on-demand instances as a backup. When doing this, it is important to position the on-demand node group after the spot node group in the `node_groups` list (since node groups with lower indices have higher priority). See [here](multi.md) for docs and examples.

There is a spot instance limit associated with your AWS account for each instance family in each region. You can check your current limit and request an increase [here](https://console.aws.amazon.com/servicequotas/home?#!/services/ec2/quotas) (set the region in the upper right corner to your desired region, type "spot" in the search bar, and click on the quota that matches your instance type). Note that the quota values indicate the number of vCPUs available, not the number of instances; different instances have a different numbers of vCPUs, which can be seen [here](https://aws.amazon.com/ec2/instance-types/).

## Example spot configuration

### Only spot instances

```yaml
node_groups:
  - name: cpu-spot
    instance_type: m5.large
    min_instances: 0
    max_instances: 5
    spot: true
    spot_config:
      instance_distribution: [m5a.large, m5d.large, m5n.large, m5ad.large, m5dn.large, m4.large, t3.large, t3a.large, t2.large]
```

### 3 on-demand base capacity with 0% on-demand above base capacity

```yaml

node_groups:
  - name: gpu-spot
    instance_type: g4dn.xlarge
    min_instances: 0
    max_instances: 5
    spot: true
    spot_config:
      on_demand_base_capacity: 3
      on_demand_percentage_above_base_capacity: 0

# instance 1-3: on-demand
# instance 4-5: spot
```

### 0 on-demand base capacity with 50% on-demand above base capacity

```yaml
node_groups:
  - name: gpu-spot
    instance_type: g4dn.xlarge
    min_instances: 0
    max_instances: 4
    spot: true
    spot_config:
      on_demand_base_capacity: 0
      on_demand_percentage_above_base_capacity: 50

# instance 1: on-demand
# instance 2: spot
# instance 3: on-demand
# instance 4: spot
```
