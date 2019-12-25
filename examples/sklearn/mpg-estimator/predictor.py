# this is an example for cortex release 0.12 and may not deploy correctly on other releases of cortex

import mlflow.sklearn
import numpy as np


class PythonPredictor:
    def __init__(self, config):
        self.model = mlflow.sklearn.load_model(config["model"])

    def predict(self, payload):
        model_input = [
            payload["cylinders"],
            payload["displacement"],
            payload["horsepower"],
            payload["weight"],
            payload["acceleration"],
        ]

        result = self.model.predict([model_input])
        return np.asscalar(result)
