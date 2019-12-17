# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import boto3
import pickle
import re

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config):
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "model.pkl")

        self.model = pickle.load(open("model.pkl", "rb"))

    def predict(self, payload):
        measurements = [
            [
                sample["sepal_length"],
                sample["sepal_width"],
                sample["petal_length"],
                sample["petal_width"],
            ]
            for sample in payload
        ]

        label_ids = self.model.predict(measurements)
        return [labels[label_id] for label_id in label_ids]
