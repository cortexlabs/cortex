# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

from summarizer import Summarizer


class PythonPredictor:
    def __init__(self, config):
        self.model = Summarizer()

    def predict(self, payload):
        return self.model(payload["text"])
