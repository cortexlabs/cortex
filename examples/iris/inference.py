import numpy


def preprocess(sample, input_metadata):
    return {
        input_metadata[0].name: numpy.asarray(
            [
                [
                    sample["sepal_length"],
                    sample["sepal_width"],
                    sample["petal_length"],
                    sample["petal_width"],
                ]
            ],
            dtype=numpy.float32,
        )
    }


def postprocess(prediction, output_metadata):
    iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]
    predicted_class_id = int(numpy.argmax(prediction[0][0]))
    return {
        "class_label": iris_labels[predicted_class_id],
        "class_index": predicted_class_id,
        "probabilites": prediction[0][0].tolist(),
    }
