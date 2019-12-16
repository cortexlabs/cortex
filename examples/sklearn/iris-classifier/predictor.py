# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import boto3
import pickle
import re


class PythonPredictor:
    def __init__(self, config):
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "model.pkl")

        self.model = pickle.load(open("model.pkl", "rb"))
        self.labels = ["setosa", "versicolor", "virginica"]

    def predict(self, payload):
        measurements = [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]

        label_id = self.model.predict([measurements])[0]
        return self.labels[label_id]
