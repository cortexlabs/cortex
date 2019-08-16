labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]


def post_inference(prediction, metadata):
    label_index = int(prediction["response"]["class_ids"][0])
    return labels[label_index]
