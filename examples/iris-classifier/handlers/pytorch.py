import numpy as np

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


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
    predicted_class_id = int(np.argmax(prediction[0][0]))
    return {
        "class_label": labels[predicted_class_id],
        "class_index": predicted_class_id,
        "probabilites": prediction[0][0],
    }
