# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`


import datadog


class PythonPredictor:
    def __init__(self, config):
        ...

    def predict(self, payload):
        datadog.statsd.increment("language_identifier", value=1, tags=["a:b", "c:d"])
