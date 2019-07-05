iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]


def post_inference(prediction, metadata):
    predicted_class_id = prediction[0][0]
    return {
        "class_label": iris_labels[predicted_class_id],
        "class_index": predicted_class_id,
        "probabilities": prediction[1][0],
    }
