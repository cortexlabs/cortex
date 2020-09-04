# Using GPUs

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

To use GPUs:

1. Make sure your AWS account is subscribed to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM).
2. You may need to [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.
3. Set instance type to an AWS GPU instance (e.g. `g4dn.xlarge`) when installing Cortex.
4. Set the `gpu` field in the `compute` configuration for your API. One unit of GPU corresponds to one virtual GPU. Fractional requests are not allowed.

## Tips

### If using `processes_per_replica` > 1, TensorFlow-based models, and Python Predictor

When using `processes_per_replica` > 1 with TensorFlow-based models (including Keras) in the Python Predictor, loading the model in separate processes at the same time will throw a `CUDA_ERROR_OUT_OF_MEMORY: out of memory` error. This is because the first process that loads the model will allocate all of the GPU's memory and leave none to other processes. To prevent this from happening, the per-process GPU memory usage can be limited. There are two methods:

1\) Configure the model to allocate only as much memory as it requires, via [tf.config.experimental.set_memory_growth()](https://www.tensorflow.org/api_docs/python/tf/config/experimental/set_memory_growth):

```python
for gpu in tf.config.list_physical_devices("GPU"):
    tf.config.experimental.set_memory_growth(gpu, True)
```

2\) Impose a hard limit on how much memory the model can use, via [tf.config.set_logical_device_configuration()](https://www.tensorflow.org/api_docs/python/tf/config/set_logical_device_configuration):

```python
mem_limit_mb = 1024
for gpu in tf.config.list_physical_devices("GPU"):
    tf.config.set_logical_device_configuration(
        gpu, [tf.config.LogicalDeviceConfiguration(memory_limit=mem_limit_mb)]
    )
```

See the [TensorFlow GPU guide](https://www.tensorflow.org/guide/gpu) and this [blog post](https://medium.com/@starriet87/tensorflow-2-0-wanna-limit-gpu-memory-10ad474e2528) for additional information.
