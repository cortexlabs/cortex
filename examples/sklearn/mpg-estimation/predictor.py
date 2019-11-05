from joblib import load
import numpy as np


model = None


def init(model_path, metadata):
    global model
    model = load(model_path)


def predict(sample, metadata):
    input_array = [
        sample["cylinders"],
        sample["displacement"],
        sample["horsepower"],
        sample["weight"],
        sample["acceleration"],
    ]

    result = model.predict([input_array])
    return np.asscalar(result)
