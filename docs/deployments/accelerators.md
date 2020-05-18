# Using Accelerators

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Cortex

To use Accelerators (Inferentia):

2. You may need to [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.
3. Set instance type to an AWS Inferentia instance (e.g. `inf1.xlarge`) when installing Cortex.
4. Set the `accelerator` field in the `compute` configuration for your API. One unit of Accelerator corresponds to one virtual Accelerator. Fractional requests are not allowed.

## Neuron

Currently, only 2 frameworks can be used with accelerators (Inferentia chips): TensorFlow and PyTorch. See the [TensorFlow guide](https://github.com/aws/aws-neuron-sdk/blob/master/docs/tensorflow-neuron/tutorial-compile-infer.md) and the [PyTorch guide](https://github.com/aws/aws-neuron-sdk/blob/master/docs/pytorch-neuron/tutorial-compile-infer.md). What is important in these guides is the compilation phase. All the rest is handled by the Cortex runtime. There are 2 examples implemented with Cortex for both frameworks [here for TensorFlow](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow/image-classifier-resnet50) and [here for PyTorch](https://github.com/cortexlabs/cortex/tree/master/examples/pytorch/image-classifier-resnet50). Use these Cortex examples to model your own project.

To better understand how Accelerators (Inferentia chips) work, read these [technical notes](https://github.com/aws/aws-neuron-sdk/blob/master/docs/technotes/README.md). For further information on what options the compiler can accept, check out this [page](https://github.com/aws/aws-neuron-sdk/tree/master/docs/neuron-cc).