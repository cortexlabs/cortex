import boto3
import numpy as np
import pickle
import re


class Predictor:
    def __init__(self, metadata):
        bucket, key = re.match("s3://(.+?)/(.+)", metadata["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "model.pkl")

        self.model = pickle.load(open("model.pkl", "rb"))
        self.labels = ["setosa", "versicolor", "virginica"]

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

        label_ids = self.model.predict(np.array(measurements))
        return [self.labels[label_id] for label_id in label_ids]
