import mlflow.sklearn
import numpy as np


class Predict:
    def __init__(self, metadata):
        self.model = mlflow.sklearn.load_model(metadata["model"])

    def predict(self, payload):
        input_array = [
            payload["cylinders"],
            payload["displacement"],
            payload["horsepower"],
            payload["weight"],
            payload["acceleration"],
        ]

        result = self.model.predict([input_array])
        return np.asscalar(result)
