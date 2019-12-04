import mlflow.sklearn
import numpy as np


model = None


def init(model_path, metadata):
    global model
    model = mlflow.sklearn.load_model(model_path)


def predict(payload, metadata):
    input_array = [
        payload["cylinders"],
        payload["displacement"],
        payload["horsepower"],
        payload["weight"],
        payload["acceleration"],
    ]

    result = model.predict([input_array])
    return np.asscalar(result)
