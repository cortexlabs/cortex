# Instances

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can spin up a Cortex cluster on a variety of AWS instance types. If you are unsure about which instance to pick, review these options as a starting point. This is not a comprehensive guide so please refer to the [full documentation](https://aws.amazon.com/ec2/instance-types/) on AWS for more information.

## T3 instances

[T3 instances](https://aws.amazon.com/ec2/instance-types/t3/) are useful for **development** clusters that primarily run model inferences with low compute and memory resource utilization.

* Compute:  low
* Memory:   low
* AWS cost: starts at $0.0052 per hour&ast;

Relevant example: [iris classification](../../examples/sklearn/iris-classifier) using scikit-learn.

## M5 instances

[M5 instances](https://aws.amazon.com/ec2/instance-types/m5/) are useful for clusters that primarily run model inferences with low compute and memory resource utilization.

* Compute:  low
* Memory:   low
* AWS cost: starts at $0.096 per hour&ast;

Relevant example: [MPG estimation](../../examples/sklearn/mpg-estimator) using scikit-learn.

## G4 instances

[G4 instances](https://aws.amazon.com/ec2/instance-types/g4/) are useful for clusters that primarily run model inferences with high compute and low memory resource utilization.

* Compute:  high
* Memory:   low
* AWS cost: starts at $0.526 per hour&ast;

Relevant example: [language identification](../../examples/pytorch/language-identifier) using PyTorch.

## P2 instances

[P2 instances](https://aws.amazon.com/ec2/instance-types/p2/) are useful for clusters that primarily run model inferences with high compute and high memory resource utilization.

* Compute:  high
* Memory:   high
* AWS cost: starts at $0.900 per hour&ast;

Relevant example: [text generation](../../examples/tensorflow/text-generator) using TensorFlow.

<br>

&ast; pricing is for the US West (Oregon) AWS region
