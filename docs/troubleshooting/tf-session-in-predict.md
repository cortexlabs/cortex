# Using TensorFlow session in predict method

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

When doing inferences with TensorFlow using the [Realtime API Python Predictor](../deployments/realtime-api/predictors.md#python-predictor) or [Batch API Python Predictor](../deployments/batch-api/predictors.md#python-predictor), it should be noted that your Python Predictor's `__init__()` constructor is only called on one thread, whereas its `predict()` method can run on any of the available threads (which is configured via the `threads_per_process` field in the API's `predictor` configuration). If `threads_per_process` is set to `1` (the default value), then there is no concern, since `__init__()` and `predict()` will run on the same thread. However, if `threads_per_process` is greater than `1`, then only one of the inference threads will have executed the `__init__()` function. This can cause issues with TensorFlow because the default graph is a property of the current thread, so if `__init__()` initializes the TensorFlow graph, only the thread that executed `__init__()` will have the default graph set.

The error you may see if the default graph is not set (as a consequence of `__init__()` and `predict()` running in separate threads) is:

```text
TypeError: Cannot interpret feed_dict key as Tensor: Tensor Tensor("Placeholder:0", shape=(1, ?), dtype=int32) is not an element of this graph.
```

To avoid this error, you can set the default graph before running the prediction in the `predict()` method:

```python

def predict(self, payload):
    with self.sess.graph.as_default():
        # perform your inference here
```
