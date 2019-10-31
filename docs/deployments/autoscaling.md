# Autoscaling

Cortex autoscales your webservices within the boundaries you specify.

## Autoscaling Replicas

Cortex adjusts the number of replicas that are serving predictions by monitoring the compute resource usage of each API. The number of replicas will be at least `min_replicas` and no more than `max_replicas`.

## Autoscaling Nodes

Cortex spins up and down nodes based on the aggregate resource requests of all APIs. The number of nodes will be at least `min_instances` and no more than `max_instances` (configured during installation and modifiable via the [AWS console](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-manual-scaling.html)).
