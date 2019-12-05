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
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]

        label_id = self.model.predict(np.array([measurements]))[0]
        return self.labels[label_id]
