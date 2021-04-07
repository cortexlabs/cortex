import os
import boto3
from botocore import UNSIGNED
from botocore.client import Config
import pickle

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config, proto_module_pb2):
        s3 = boto3.client("s3")
        s3.download_file(config["bucket"], config["key"], "/tmp/model.pkl")
        self.model = pickle.load(open("/tmp/model.pkl", "rb"))
        self.proto_module_pb2 = proto_module_pb2

    def predict(self, payload):
        measurements = [
            payload.sepal_length,
            payload.sepal_width,
            payload.petal_length,
            payload.petal_width,
        ]

        label_id = self.model.predict([measurements])[0]
        return self.proto_module_pb2.Response(classification=labels[label_id])
