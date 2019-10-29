labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def post_inference(prediction, signature, metadata):
    predicted_class_id = int(prediction["class_ids"][0])
    return labels[predicted_class_id]
