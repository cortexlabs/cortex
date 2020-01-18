# Spot instances

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

[Spot instances](https://aws.amazon.com/ec2/spot/) are spare capacity that AWS sells at a discount (up to 90%). The caveat is that spot instances may not always be available, and can be recalled by AWS at anytime. Cortex allows you to use spot instances in your cluster to take advantage of the discount while ensuring uptime and reliability of APIs. You can configure your cluster to use spot instances using the configuration below:

```yaml
# cluster.yaml

# whether to use spot instances in the cluster; spot instances are not guaranteed to be available so please take that into account for production clusters (default: false)
spot: false

spot_config:
  # additional instances with identical or better specs than the primary instance type (defaults to 2 instances sorted by price)
  instance_distribution: [similar_instance_type_1, similar_instance_type_2]

  # minimum number of on demand instances (default: 0)
  on_demand_base_capacity: 0

  # percentage of on demand instances to use after the on demand base capacity has been met [0, 100] (default: 50)
  # note: setting this to 0 may hinder cluster scale up when spot instances are not available
  on_demand_percentage_above_base_capacity: 0

  # max price for spot instances (default: the on-demand price of the primary instance type)
  max_price: <float>

  # number of spot instance pools across which to allocate spot instances [1, 20] (default: number of instances in instance distribution)
  instance_pools: 3

  # fallback to on-demand instances if spot instances were unable to be allocated (default: true)
  on_demand_backup: true
```

Spot instances are not guaranteed to be available. The chances of getting spot instances can be improved by providing `instance_distribution`, a list of alternative instance types to the primary `instance_type` you specified. If left blank, Cortex will autofill `instance_distribution` with up to 2 other similar instances. Cortex defaults the `max_price` to the on-demand price of the primary instance.

Spot instances can be mixed with on-demand instances by configuring `on_demand_base_capacity` and `on_demand_percentage_above_base_capacity`. `on_demand_base_capacity` enforces the minimum number of nodes that will be fulfilled by on-demand instances as your cluster is scaling up. `on_demand_percentage_above_base_capacity` defines the percentage of instances that will be on-demand after the base capacity has been fulfilled (the rest being spot instances). `instance_pools` is the number of pools per availability zone to allocate your instances from. See [here](https://docs.aws.amazon.com/autoscaling/ec2/APIReference/API_InstancesDistribution.html) for more details.

Even if multiple instances are specified in your `instance_distribution` on-demand instances are mixed, there is still a possibility of running into scale up issues when attempting to spin up spot instances. Spot instance requests may not be fulfilled for several reasons. Spot instance pricing fluctuates, therefore the `max_price` may be lower than the current spot pricing rate. Another possibility could be that the availability zones of the cluster ran out of spot instances. `on_demand_backup` can be used mitigate the impact of unfulfilled spot requests by enabling the cluster to spin up on-demand instances if spot instance requests are not fulfilled within 5 minutes.

## Example spot configuration

### Only spot instances with backup

```yaml

spot: true

spot_config:
    on_demand_base_capacity: 0
    on_demand_percentage_above_base_capacity: 0
    on_demand_backup: true # recommended for production clusters
```

### 3 on-demand base capacity with 0% on-demand above base capacity

```yaml
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
