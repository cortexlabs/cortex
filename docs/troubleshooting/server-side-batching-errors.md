# Batching errors when batch_size/batch_timeout are set

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

When `batch_size` and `batch_timeout` fields are set for the [TensorFlow Predictor](../deployments/predictors.md#tensorflow-predictor), errors can be encountered if the associated model hasn't been built for batching.

The following error is an example of what happens when the input shape doesn't accomodate batching - e.g. when its shape is `[height, width, 3]` instead of `[batch_size, height, width, 3]`.
```text
Batching session Run() input tensors must have at least one dimension.
```

The next error is another example of setting the output shape inappropriately for batching - e.g. when its shape is `[labels]` instead of `[batch_size, labels]`.

```text
Batched output tensor has 0 dimensions.
```

The solution to these 2 errors is to incorporate into the model's graph another dimension placed on the first position for both its input and output.
