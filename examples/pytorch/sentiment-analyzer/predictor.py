from fastai.text import *
import requests

class PythonPredictor:
    def __init__(self, config):
        # Download model file
        req = requests.get("https://cortex-example-projects.s3-us-west-2.amazonaws.com/export.pkl")
        with open("export.pkl", "wb") as model:
            model.write(req.content)

        # Initialize model
        self.predictor = load_learner("."")


    def predict(self, payload):
        prediction = self.predictor.predict(payload["text"])
        return prediction[0].obj
