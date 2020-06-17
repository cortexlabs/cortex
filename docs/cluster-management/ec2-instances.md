# EC2 instances

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

There are a variety of instance types to choose from when creating a Cortex cluster. If you are unsure about which instance to pick, review these options as a starting point.

This is not a comprehensive guide so please refer to the [AWS's documentation](https://aws.amazon.com/ec2/instance-types/) for more information.

Note: you may have limited (or no) access to certain instance types. To check your limits, click [here](https://console.aws.amazon.com/ec2/v2/home?#Limits:), set your region in the upper right, and type "on-demand" in the search box. You can request a limit by selecting an instance family and clicking "Request limit increase" in the upper right. Note that the limits are vCPU-based no matter the instance type (e.g. to run 4 `g4dn.xlarge` instances, you will need a 16 vCPU limit for G instances).

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
