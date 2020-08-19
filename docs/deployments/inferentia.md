# Using Inferentia

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

To use [Inferentia ASICs](https://aws.amazon.com/machine-learning/inferentia/):

1. You may need to [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.
1. Set the instance type to an AWS Inferentia instance (e.g. `inf1.xlarge`) when creating your Cortex cluster.
1. Set the `inf` field in the `compute` configuration for your API. One unit of `inf` corresponds to one Inferentia ASIC with 4 NeuronCores *(not the same thing as `cpu`)* and 8GB of cache memory *(not the same thing as `mem`)*. Fractional requests are not allowed.

## Neuron

Inferentia ASICs come in different sizes depending on the instance type:

* `inf1.xlarge`/`inf1.2xlarge` - each has 1 Inferentia ASIC
* `inf1.6xlarge` - has 4 Inferentia ASICs
* `inf1.24xlarge` - has 16 Inferentia ASICs

Each Inferentia ASIC comes with 4 NeuronCores and 8GB of cache memory. To better understand how Inferentia ASICs work, read these [technical notes](https://github.com/aws/aws-neuron-sdk/blob/master/docs/technotes/README.md) and this [FAQ](https://github.com/aws/aws-neuron-sdk/blob/master/FAQ.md).

### NeuronCore Groups

A [NeuronCore Group](https://github.com/aws/aws-neuron-sdk/blob/master/docs/tensorflow-neuron/tutorial-NeuronCore-Group.md) (NCG) is a set of NeuronCores that is used to load and run a compiled model. NCGs exist to aggregate NeuronCores to improve hardware performance. Models can be shared within an NCG, but this would require the device driver to dynamically context switch between each model, which degrades performance. Therefore we've decided to only allow one model per NCG (unless you are using a [multi-model endpoint](../guides/multi-model.md), in which case there will be multiple models on a single NCG, and there will be context switching).

Each Cortex API process will have its own copy of the model and will run on its own NCG (the number of API processes is configured by the [`processes_per_replica`](realtime-api/autoscaling.md#replica-parallelism) for Realtime APIs field in the API configuration). Each NCG will have an equal share of NeuronCores. Therefore, the size of each NCG will be `4 * inf / processes_per_replica` (`inf` refers to your API's `compute` request, and it's multiplied by 4 because there are 4 NeuronCores per Inferentia chip).

For example, if your API requests 2 `inf` chips, there will be 8 NeuronCores available. If you set `processes_per_replica` to 1, there will be one copy of your model running on a single NCG of size 8 NeuronCores. If `processes_per_replica` is 2, there will be two copies of your model, each running on a separate NCG of size 4 NeuronCores. If `processes_per_replica` is 4, there will be 4 NCGs of size 2 NeuronCores, and if If `processes_per_replica` is 8, there will be 8 NCGs of size 1 NeuronCores. In this scenario, these are the only valid values for `processes_per_replica`. In other words the total number of requested NeuronCores (which equals 4 * the number of requested Inferentia chips) must be divisible by `processes_per_replica`.

The 8GB cache memory is shared between all 4 NeuronCores of an Inferentia chip. Therefore an NCG with 8 NeuronCores (i.e. 2 Inf chips) will have access to 16GB of cache memory. An NGC with 2 NeuronCores will have access to 8GB of cache memory, which will be shared with the other NGC of size 2 running on the same Inferentia chip.

### Compiling models

Before a model can be deployed on Inferentia chips, it must be compiled for Inferentia. The Neuron compiler can be used to convert a regular TensorFlow SavedModel or PyTorch model into the hardware-specific instruction set for Inferentia. Inferentia currently supports compiled models from TensorFlow and PyTorch.

By default, the Neuron compiler will compile a model to use 1 NeuronCore, but can be manually set to a different size (1, 2, 4, etc).

For optimal performance, your model should be compiled to run on the number of NeuronCores available to it. The number of NeuronCores will be `4 * inf / processes_per_replica` (`inf` refers to your API's `compute` request, and it's multiplied by 4 because there are 4 NeuronCores per Inferentia chip). See [NeuronCore Groups](#neuron-core-groups) above for an example, and see [Improving performance](#improving-performance) below for a discussion of choosing the appropriate number of NeuronCores.

Here is an example of compiling a TensorFlow SavedModel for Inferentia:

```python
import tensorflow.neuron as tfn

tfn.saved_model.compile(
    model_dir,
    compiled_model_dir,
    batch_size,
    compiler_args=["--num-neuroncores", "1"],
)
```

Here is an example of compiling a PyTorch model for Inferentia:

```python
import torch_neuron, torch

model.eval()
example_input = torch.zeros([batch_size] + input_shape, dtype=torch.float32)
model_neuron = torch.neuron.trace(
    model,
    example_inputs=[example_input],
    compiler_args=["--num-neuroncores", "1"]
)
model_neuron.save(compiled_model)
```

The versions of `tensorflow-neuron` and `torch-neuron` that are used by Cortex are found in the [Realtime API pre-installed packages list](realtime-api/predictors.md#inferentia-equipped-apis) and [Batch API pre-installed packages list](batch-api/predictors.md#inferentia-equipped-apis). When installing these packages with `pip` to compile models of your own, use the extra index URL `--extra-index-url=https://pip.repos.neuron.amazonaws.com`.

See AWS's [TensorFlow](https://github.com/aws/aws-neuron-sdk/blob/master/docs/tensorflow-neuron/tutorial-compile-infer.md#step-3-compile-on-compilation-instance) and [PyTorch](https://github.com/aws/aws-neuron-sdk/blob/master/docs/pytorch-neuron/tutorial-compile-infer.md#step-3-compile-on-compilation-instance) guides on how to compile models for Inferentia. Here are 2 examples implemented with Cortex:

<!-- CORTEX_VERSION_MINOR x2 -->
1. [ResNet50 in TensorFlow](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow/image-classifier-resnet50)
1. [ResNet50 in PyTorch](https://github.com/cortexlabs/cortex/tree/master/examples/pytorch/image-classifier-resnet50)

### Improving performance

A few things can be done to improve performance using compiled models on Cortex:

1. There's a minimum number of NeuronCores for which a model can be compiled. That number depends on the model's architecture. Generally, compiling a model for more cores than its required minimum helps to distribute the model's operators across multiple cores, which in turn [can lead to lower latency](https://github.com/aws/aws-neuron-sdk/blob/master/docs/technotes/neuroncore-pipeline.md). However, compiling a model for more NeuronCores means that you'll have to set `processes_per_replica` to be lower so that the NeuronCore Group has access to the number of NeuronCores for which you compiled your model. This is acceptable if latency is your top priority, but if throughput is more important to you, this tradeoff is usually not worth it. To maximize throughput, compile your model for as few NeuronCores as possible and increase `processes_per_replica` to the maximum possible (see above for a sample calculation).

1. Try to achieve a near [100% placement](https://github.com/aws/aws-neuron-sdk/blob/b28262e3072574c514a0d72ad3fe5ca48686d449/src/examples/tensorflow/keras_resnet50/pb2sm_compile.py#L59) of your model's graph onto the NeuronCores. During the compilation phase, any operators that can't execute on NeuronCores will be compiled to execute on the machine's CPU and memory instead. Even if just a few percent of the operations reside on the host's CPU/memory, the maximum throughput of the instance can be significantly limited.

1. Use the [`--static-weights` compiler option](https://github.com/aws/aws-neuron-sdk/blob/master/docs/technotes/performance-tuning.md#compiling-for-pipeline-optimization) when possible. This option tells the compiler to make it such that the entire model gets cached onto the NeuronCores. This avoids a lot of back-and-forth between the machine's CPU/memory and the Inferentia ASICs.
