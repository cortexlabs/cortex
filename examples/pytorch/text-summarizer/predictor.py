# this is an example for cortex release 0.14 and may not deploy correctly on other releases of cortex

from summarizer import Summarizer


class PythonPredictor:
    def __init__(self, config):
        self.model = Summarizer()

    def predict(self, payload):
        return self.model(payload["text"])
