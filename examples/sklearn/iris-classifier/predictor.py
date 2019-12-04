import pickle
import numpy as np


model = None
labels = ["setosa", "versicolor", "virginica"]


def init(model_path, metadata):
    global model
    model = pickle.load(open(model_path, "rb"))


def predict(payload, metadata):
    measurements = [
        payload["sepal_length"],
        payload["sepal_width"],
        payload["petal_length"],
        payload["petal_width"],
    ]

    label_id = model.predict(np.array([measurements]))[0]
    return labels[label_id]
