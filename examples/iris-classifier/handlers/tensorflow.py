labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def post_inference(prediction, metadata):
    predicted_class_id = int(prediction["response"]["class_ids"][0])
    return labels[predicted_class_id]
