import numpy as np

iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]


def preprocess(sample, input_metadata):
    return {
        input_metadata[0].name: np.asarray(
            [
                [
                    sample["sepal_length"],
                    sample["sepal_width"],
                    sample["petal_length"],
                    sample["petal_width"],
                ]
            ],
            dtype=np.float32,
        )
    }


def postprocess(prediction, output_metadata):
    predicted_class_id = int(np.argmax(prediction[0][0]))
    return {
        "class_label": iris_labels[predicted_class_id],
        "class_index": predicted_class_id,
        "probabilites": prediction[0][0].tolist(),
    }
