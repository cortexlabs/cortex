import numpy as np

iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]


def preinference(request, metadata):
    return {
        metadata[0].name: [
            request["sepal_length"],
            request["sepal_width"],
            request["petal_length"],
            request["petal_width"],
        ]
    }


def postinference(response, metadata):
    predicted_class_id = int(np.argmax(response[0][0]))
    return {
        "class_label": iris_labels[predicted_class_id],
        "class_index": predicted_class_id,
        "probabilites": response[0][0],
    }
