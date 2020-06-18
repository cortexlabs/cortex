# TensorFlow session called in predict method

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Context

When doing inferences with TensorFlow using the Python Predictor, it has to be noted that Python Predictor constructor and its `predict` method run on different threads. This means that when the program enters the `predict` method, the current session which has presumably been saved as an attribute inside the [`PythonPredictor`](../deployments/predictors.md#python-predictor) object won't point to the default graph the session has been initialized with.

The error you will get as a consequence of having run the 2 methods (constructor and `predict` method) in different threads is:
`
TypeError: Cannot interpret feed_dict key as Tensor: Tensor Tensor("Placeholder:0", shape=(1, ?), dtype=int32) is not an element of this graph.
`

## Use session.graph.as_default()

For this error to be avoided, you need to set the default graph and session right before running the prediction in the `predict` method:
```python

def predict(self, payload):
    with self.sess.graph.as_default():
        # your implementation of predict_on_session
        self.prediction = predict_on_sess(sess, payload)
    return self.prediction
```

## Is it slow?

It has been observed that calling `self.session.graph.as_default()` takes about *0.4 microseconds* on average. This works for any value of `threads_per_worker`.

The following only applies when `threads_per_worker` is set to 1. Use this when you want to have a minimal computational cost. For this, you can have a separate method that loads the default session and graph once for the given thread. Here's one approach:

```python
class PythonPredictor:
    def __init__(self, config):
        self.sess = tf.Session(...)
        self.default_graph_loaded = False

    def load_graph_if_not_present(self):
        """
        Placeholder for "with self.sess.graph.as_default()"

        Relevant sources of information:
        https://stackoverflow.com/a/49468139/2096747
        https://stackoverflow.com/a/57397201/2096747
        https://github.com/tensorflow/tensorflow/blob/master/tensorflow/python/client/session.py#L1591-L1601
        """
        if not self.default_graph_loaded:
            self.sess.__enter__()
            self.default_graph_loaded = True

    def predict(self, payload):
        self.load_graph_if_not_present()
        # your implementation of predict_on_session
        prediction = predict_on_sess(self.sess, payload)
        return prediction
```

The above implementation calls `__enter__()` method only once and after that it only checks if a flag is set. The computational cost for this is minimal, and therefore, a higher throughput could be achieved. This implementation is about 2 orders of magnitude faster than the other one.
