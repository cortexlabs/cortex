import numpy as np
import random
from flask import Flask, request
from cortex.lib.log import get_logger
import random

logger = get_logger()

iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]


def pre_inference(sample, metadata):
    return [
        sample["sepal_length"],
        sample["sepal_width"],
        sample["petal_length"],
        sample["petal_width"],
    ]


def post_inference(prediction, metadata):
    logger.info(request.path)
    predicted_class_id = int(random.random() * 3)
    return {"class_label": iris_labels[predicted_class_id], "class_index": predicted_class_id}
