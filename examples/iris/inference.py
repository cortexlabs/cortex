import numpy as np

iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]


def preinference(request, metadata):
    return {
        "input": [
            request["sepal_length"],
            request["sepal_width"],
            request["petal_length"],
            request["petal_width"],
        ]
    }


def postinference(response, metadata):
    predicted_class_id = response[0][0]
    return {"class_label": iris_labels[predicted_class_id], "class_index": predicted_class_id}
