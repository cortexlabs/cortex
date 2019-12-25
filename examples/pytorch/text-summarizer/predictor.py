# this is an example for cortex release 0.12 and may not deploy correctly on other releases of cortex your `cortex version`

from summarizer import Summarizer


class PythonPredictor:
    def __init__(self, config):
        self.model = Summarizer()

    def predict(self, payload):
        return self.model(payload["text"])
