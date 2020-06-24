# TensorFlow session called in predict method

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Context

When doing inferences with TensorFlow using the [Python Predictor](../deployments/predictors.md#python-predictor), it has to be noted that Python Predictor constructor and its `predict` method run on different threads. This means that when the program enters the `predict` method, the current session which has presumably been saved as an attribute inside the [`PythonPredictor`](../deployments/predictors.md#python-predictor) object won't point to the default graph the session has been initialized with.

The error you will get as a consequence of having run the 2 methods (constructor and `predict` method) in different threads is:
`
TypeError: Cannot interpret feed_dict key as Tensor: Tensor Tensor("Placeholder:0", shape=(1, ?), dtype=int32) is not an element of this graph.
`

## Use one-threaded processes

When `threads_per_process` is set to 1 and `processes_per_replica` >= 1, both the constructor and the `predict` method of Python Predictor run on the same thread, meaning that the above error won't occur anymore. The downside to this is that the API replica is limited to 1 thread per process.

For it to work with multiple threads, check the following section.

## Use session.graph.as_default()

For this error to be avoided on any number of threads, you need to set the default graph and session right before running the prediction in the `predict` method:
```python

def predict(self, payload):
    with self.sess.graph.as_default():
        # your implementation of predict_on_session
        self.prediction = predict_on_sess(sess, payload)
    return self.prediction
```
