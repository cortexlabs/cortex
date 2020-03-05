# this is an example for cortex release 0.14 and may not deploy correctly on other releases of cortex

from fastai.text import *
import requests


class PythonPredictor:
    def __init__(self, config):
        req = requests.get(
            "https://cortex-examples.s3-us-west-2.amazonaws.com/pytorch/sentiment-analyzer/export.pkl"
        )
        with open("export.pkl", "wb") as model:
            model.write(req.content)

        self.predictor = load_learner(".")

    def predict(self, payload):
        prediction = self.predictor.predict(payload["text"])
        return prediction[0].obj
