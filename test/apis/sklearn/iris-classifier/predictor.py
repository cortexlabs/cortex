import os
import time
import boto3
from botocore import UNSIGNED
from botocore.client import Config
import pickle

import iris_classifier_pb2

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config):
        if os.environ.get("AWS_ACCESS_KEY_ID"):
            s3 = boto3.client("s3")  # client will use your credentials if available
        else:
            s3 = boto3.client("s3", config=Config(signature_version=UNSIGNED))  # anonymous client

        s3.download_file(config["bucket"], config["key"], "/tmp/model.pkl")
        self.model = pickle.load(open("/tmp/model.pkl", "rb"))

    def predict(self, payload):
        measurements = [
            payload.sepal_length,
            payload.sepal_width,
            payload.petal_length,
            payload.petal_width,
        ]

        time.sleep(1)

        label_id = self.model.predict([measurements])[0]
        return iris_classifier_pb2.Response(classification=labels[label_id])
