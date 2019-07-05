import numpy as np

iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]


def pre_inference(sample, metadata):
    return {
        "input": [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]
    }


def post_inference(prediction, metadata):
    predicted_class_id = prediction[0][0]
    return {"class_label": iris_labels[predicted_class_id], "class_index": predicted_class_id}
