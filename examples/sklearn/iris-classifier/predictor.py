import pickle

model = None


def init(model_path, metadata):
    global model
    model = pickle.load(open(model_path, "rb"))


import numpy

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def predict(sample, metadata):
    input_array = numpy.array(
        [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    )

    label_id = model.predict([input_array])[0]
    return labels[label_id]
