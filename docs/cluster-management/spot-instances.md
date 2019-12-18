# Spot Instances

[Spot Instances](https://aws.amazon.com/ec2/spot/) are spare capacity that AWS sells at a discount (up to 90%). The caveat is that Spot instances can be recalled by AWS at anytime. Cortex allows you to use Spot Instances in your cluster to take advantage of the discount while ensuring uptime and reliability of deployments. You can enable and configure your cluster to use Spot Instances using the configuration below.

```yaml
# whether to use spot instances in the cluster (default: false)
# spot instances are not guaranteed to be available so please take that into account for production clusters
spot: false

spot_config:
  # additional instances with identical or better specs than the primary instance type (defaults to 2 instances sorted by price)
  instance_distribution: # [t3.large, t3a.large] (example distribution for m5.xlarge)

  # minimum number of on demand instances (default: 0)
  on_demand_base_capacity: 0

  # percentage of on demand instances to use after the on demand base capacity has been met [0, 100] (default: 50)
  # note: setting this to 0 may hinder cluster scale up when spot instances are not available
  on_demand_percentage_above_base_capacity: 50

  # max price for instances (defaults to the on demand price of the primary instance type)
  max_price: # 0.096 (example max price of m5.xlarge)

  # number of spot instance pools across which to allocate spot instances [1, 20] (default: number of instances in instance distribution)
  instance_pools: 2
```

Since spot instances are spare capacity, they are not guaranteed to be available. The chances of getting spot instances can be improved by providing `instance_distribution`, a list of instances that are identical or better than the primary `instance_type` you specified in your `cluster.yaml` or in prompts. If you leave it blank, Cortex will autofill `instance_distribution` with up to 2 other compatible instances. Cortex defaults the `max_price` to the on-demand price of the primary instance to make sure that you don't pay more than the on-demand price of the primary instance. Even if multiple instances are specified in your instance_distribution, there is still a possibility of not getting spot instances. **If a spot instances request can not be fulfilled, an on-demand instance is not requested by default. The cluster will fail to scale up.** To ensure uptime and reliability on production workloads, please configure `on_demand_base_capacity` and `on_demand_percentage_above_base_capacity` appropriately. `on_demand_base_capacity` enforces the minimum number of nodes that will be fulfilled by on-demand instances as your cluster is scaling up. `on_demand_percentage_above_base_capacity` defines the percentage of instances that will be on-demand after the base capacity has been fulfilled (the rest being spot). `instance_pools` is the number of pools per availability zone to allocate your instances from. Cortex defaults this configuration to the length `instance_distribution` (including the primary instance). See [here](https://docs.aws.amazon.com/autoscaling/ec2/APIReference/API_InstancesDistribution.html) for more details.

## Example spot configuration

### Only Spot Instances
```yaml
# not recommended for production workloads

spot: true

spot_config:
    on_demand_base_capacity: 0
    on_demand_percentage_above_base_capacity: 0
```

### 60% On-Demand base
```yaml
min_instances: 0
max_instances: 5

spot: true
spot_config:
    on_demand_base_capacity: 3
    on_demand_percentage_above_base_capacity: 0

# instances 1-3: On-Demand
# instances 4-5: Spot
```

### 0% On-Demand base with 50% On-Demand above base
```yaml
min_instances: 0
max_instances: 4

spot: true
spot_config:
    on_demand_base_capacity: 0
    on_demand_percentage_above_base_capacity: 50

# instance 1: On-Demand
# instance 2: Spot
# instance 3: On-Demand
# instance 4: Spot
```
