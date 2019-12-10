from summarizer import Summarizer


class Predictor:
    def __init__(self, config):
        self.model = Summarizer()

    def predict(self, payload):
        return self.model(payload["text"])
