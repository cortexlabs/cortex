# EC2 instances

You can spin up a Cortex cluster on a variety of AWS instance types. If you are unsure about which instance to pick, review these options as a starting point. This is not a comprehensive guide so please refer to the [full documentation](https://aws.amazon.com/ec2/instance-types/) on AWS for more information.

## T3 instances

[T3 instances](https://aws.amazon.com/ec2/instance-types/t3/) are useful for **development** clusters that primarily run model inferences with low compute and memory resource utilization.

* Example: iris classification with scikit-learn
* Compute: low
* Memory: low
* GPU: no
* AWS cost: starts at $0.0052 per hour&ast;

## C5 instances

[C5 instances](https://aws.amazon.com/ec2/instance-types/c5/) are useful for clusters that primarily run model inferences with medium compute and low memory resource utilization.

* Example: language identification with fastText
* Compute: medium
* Memory: low
* GPU: no
* AWS cost: starts at $0.085 per hour&ast;

## M5 instances

[M5 instances](https://aws.amazon.com/ec2/instance-types/m5/) are useful for clusters that primarily run model inferences with low compute and memory resource utilization.

* Example: MPG estimation with scikit-learn
* Compute: low
* Memory: medium
* GPU: no
* AWS cost: starts at $0.096 per hour&ast;

## G4 instances

[G4 instances](https://aws.amazon.com/ec2/instance-types/g4/) are useful for clusters that primarily run model inferences with high compute and low memory resource utilization that can run on GPUs.

* Example: image classification with Inception
* Compute: high
* Memory: medium
* GPU: yes
* AWS cost: starts at $0.526 per hour&ast;

## P2 instances

[P2 instances](https://aws.amazon.com/ec2/instance-types/p2/) are useful for clusters that primarily run model inferences with high compute and high memory resource utilization that can run on GPUs.

* Example: text generation with GPT-2
* Compute: high
* Memory: high
* GPU: yes
* AWS cost: starts at $0.900 per hour&ast;

<br>

&ast; On-demand pricing for the US West (Oregon) AWS region.
