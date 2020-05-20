# Using ASICs

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Cortex

To use ASICs (Inferentia):

2. You may need to [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.
3. Set instance type to an AWS Inferentia instance (e.g. `inf1.xlarge`) when installing Cortex.
4. Set the `asic` field in the `compute` configuration for your API. One unit of ASIC corresponds to one virtual ASIC. Fractional requests are not allowed.

## Neuron

Cortex supports one type of ASICs: [AWS Inferentia ASICs](https://aws.amazon.com/machine-learning/inferentia/).

These ASICs come in different sizes depending on the instance type:

* `inf1.xlarge`/`inf1.2xlarge` - each has 1 Inferentia chip.
* `inf1.6xlarge` - has 4 Inferentia chips.
* `inf1.24xlarge` - has 16 Inferentia chips.

Each Inferentia chip comes with 4 Neuron Cores and 8GB of cache memory. To better understand how ASICs (Inferentia) work, read these [technical notes](https://github.com/aws/aws-neuron-sdk/blob/master/docs/technotes/README.md) and this [FAQ](https://github.com/aws/aws-neuron-sdk/blob/master/FAQ.md).

### NeuronCore Groups

An NCG ([*NeuronCore Group*](https://github.com/aws/aws-neuron-sdk/blob/master/docs/tensorflow-neuron/tutorial-NeuronCore-Group.md)) is a set of Neuron Cores that are used to load and run a compiled model. At any point in time, only one model will be running in an NCG. Models can also be shared within an NCG, but for that to happen, the device driver is going to have to dynamically context switch between each model - therefore the Cortex team has decided to only allow one model per NCG to improve performance. The compiled output models are saved in the same format as the source's.

NCGs exist for the sole purpose of aggregating Neuron Cores to improve hardware performance. It is advised to set the NCGs' size to that of the compiled model's within your API. The NCGs' size is determined indirectly using the available number of ASIC chips to the API and the number of workers per replica. Check the [`workers_per_replica` description](autoscaling.md#replica-parallelism) to find out how the resources are partitioned.

Before a model is deployed on ASIC hardware, it first has to be compiled for the said hardware. The Neuron compiler can be used to convert a regular TF SavedModel or PyTorch model into hardware-specific instruction set for Inferentia. Cortex currently supports TensorFlow and PyTorch compiled models.

### Compiling Models

By default, the neuron compiler will try to compile a model to use 1 Neuron core, but can be manually set to a different size (1, 2, 4, etc). To understand why setting a higher Neuron core count can improve performance, read [NeuronCore Pipeline notes](https://github.com/aws/aws-neuron-sdk/blob/master/docs/technotes/neuroncore-pipeline.md).

```python
# for TensorFlow SavedModel
import tensorflow.neuron as tfn
tfn.saved_model.compile(
    model_dir,
    compiled_model_dir,
    compiler_args=["--num-neuroncores", "1"]
)

# for PyTorch model
import torch_neuron, torch
model.eval()
model_neuron = torch.neuron.trace(
    model,
    example_inputs=[example_input],
    compiler_args=["--num-neuroncores", "1"]
)
model_neuron.save(compiled_model)
```

The current versions of `tensorflow-neuron` and `torch-neuron` are found in the [pre-installed packages list](predictors.md#for-asic-equipped-apis). To compile models of your own, these packages have to installed using the extra index url for pip `--extra-index-url=https://pip.repos.neuron.amazonaws.com`.

See the [TensorFlow](https://github.com/aws/aws-neuron-sdk/blob/master/docs/tensorflow-neuron/tutorial-compile-infer.md#step-3-compile-on-compilation-instance) and the [PyTorch](https://github.com/aws/aws-neuron-sdk/blob/master/docs/pytorch-neuron/tutorial-compile-infer.md#step-3-compile-on-compilation-instance) guides on how to compile models to be used on ASIC hardware. There are 2 examples implemented with Cortex for both frameworks:

1. ResNet50 [example model](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow/image-classifier-resnet50) implemented for TensorFlow.
1. ResNet50 [example model](https://github.com/cortexlabs/cortex/tree/master/examples/pytorch/image-classifier-resnet50) implemented for PyTorch.
