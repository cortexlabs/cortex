# Batching errors when max_batch_size/batch_interval are set

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

When `max_batch_size` and `batch_interval` fields are set for the [Realtime API TensorFlow Predictor](../deployments/realtime-api/predictors.md#tensorflow-predictor), errors can be encountered if the associated model hasn't been built for batching.

The following error is an example of what happens when the input shape doesn't accommodate batching - e.g. when its shape is `[height, width, 3]` instead of `[batch_size, height, width, 3]`:

```text
Batching session Run() input tensors must have at least one dimension.
```

Here is another example of setting the output shape inappropriately for batching - e.g. when its shape is `[labels]` instead of `[batch_size, labels]`:

```text
Batched output tensor has 0 dimensions.
```

The solution to these errors is to incorporate into the model's graph another dimension (a placeholder for batch size) placed on the first position for both its input and output.

The following is an example of how the input `x` and the output `y` of the graph could be shaped to be compatible with server-side batching:

```python
batch_size = None
sample_shape = [340, 240, 3] # i.e. RGB image
output_shape = [1000] # i.e. image labels

with graph.as_default():
    # ...
    x = tf.placeholder(tf.float32, shape=[batch_size] + sample_shape, name="input")
    y = tf.placeholder(tf.float32, shape=[batch_size] + output_shape, name="output")
    # ...
```
