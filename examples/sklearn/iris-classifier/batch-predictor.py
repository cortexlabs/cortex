import pickle
import numpy as np


model = None
labels = ["setosa", "versicolor", "virginica"]


def init(model_path, metadata):
    global model
    model = pickle.load(open(model_path, "rb"))


def predict(payload, metadata):
    measurements = [
        [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
        for sample in payload
    ]

    label_ids = model.predict(np.array(measurements))
    return [labels[label_id] for label_id in label_ids]
