import numpy as np
from cortex.lib.log import get_logger
import random

logger = get_logger()

iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]


def pre_inference(sample, metadata):
    return {
        metadata[0].name: [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    }


def post_inference(prediction, metadata):
    predicted_class_id = prediction[0][0]

    return {"class_name": iris_labels[predicted_class_id], "class_label": random.random() * 200}

