# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import numpy as np

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def pre_inference(payload, signature, metadata):
    return {
        signature[0].name: [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]
    }


def post_inference(prediction, signature, metadata):
    predicted_class_id = prediction[0][0]
    return labels[predicted_class_id]
