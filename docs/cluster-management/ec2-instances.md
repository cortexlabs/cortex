# EC2 instances

There are a variety of instance types to choose from when creating a Cortex cluster. If you are unsure about which instance to pick, review these options as a starting point.

This is not a comprehensive guide so please refer to the [AWS's documentation](https://aws.amazon.com/ec2/instance-types/) for more information.

Note: There is an instance limit associated with your AWS account for each instance family in each region, for on-demand and for spot instances. You can check your current limit and request an increase [here](https://console.aws.amazon.com/servicequotas/home?#!/services/ec2/quotas) (set the region in the upper right corner to your desired region, type "on-demand" or "spot" in the search bar, and click on the quota that matches your instance type). Note that the quota values indicate the number of vCPUs available, not the number of instances; different instances have a different numbers of vCPUs, which can be seen [here](https://aws.amazon.com/ec2/instance-types/).

| Instance Type                                           | CPU    | Memory    | GPU Memory               | Starting price per hour* | Notes                             |
| :---                                                    | :---   | :---      | :---                     | :---                     | :---                              |
| [T3](https://aws.amazon.com/ec2/instance-types/t3/)     | low    | low       | -                        | $0.0416 (t3.medium)      | good for dev clusters             |
| [M5](https://aws.amazon.com/ec2/instance-types/m5/)     | medium | medium    | -                        | $0.096 (m5.large)        | standard cpu-based                |
| [C5](https://aws.amazon.com/ec2/instance-types/c5/)     | high   | medium    | -                        | $0.085 (c5.large)        | high cpu                          |
| [R5](https://aws.amazon.com/ec2/instance-types/r5/)     | medium | high      | -                        | $0.126 (r5.large)        | high memory                       |
| [G4](https://aws.amazon.com/ec2/instance-types/g4/)     | high   | high      | ~15GB (g4dn.xlarge)      | $0.526 (g4dn.xlarge)     | standard gpu-based                |
| [P2](https://aws.amazon.com/ec2/instance-types/p2/)     | high   | very high | ~12GB (p2.xlarge)        | $0.90 (p2.xlarge)        | high host memory gpu-based        |
| [Inf1](https://aws.amazon.com/ec2/instance-types/inf1/) | high   | medium    | ~8GB (inf1.xlarge)       | $0.368 (inf1.xlarge)     | very good price/performance ratio |

&ast; on-demand pricing for the US West (Oregon) AWS region.
