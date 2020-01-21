# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import boto3
import pickle

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config):
        s3 = boto3.client("s3")
        s3.download_file(config["bucket"], config["key"], "model.pkl")
        self.model = pickle.load(open("model.pkl", "rb"))

    def predict(self, payload):
        measurements = [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]

        label_id = self.model.predict([measurements])[0]
        return labels[label_id]
