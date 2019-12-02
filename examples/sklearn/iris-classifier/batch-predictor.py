import pickle
import numpy as np


model = None
labels = ["setosa", "versicolor", "virginica"]


def init(model_path, metadata):
    global model
    model = pickle.load(open(model_path, "rb"))


def predict(sample, metadata):
    measurements = [
        [s["sepal_length"], s["sepal_width"], s["petal_length"], s["petal_width"]] for s in sample
    ]

    label_ids = model.predict(np.array(measurements))
    return [labels[label_id] for label_id in label_ids]
