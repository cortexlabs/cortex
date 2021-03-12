import os
import boto3
from botocore import UNSIGNED
from botocore.client import Config
import pickle

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config):
        if os.environ.get("AWS_ACCESS_KEY_ID"):
            s3 = boto3.client("s3")  # client will use your credentials if available
        else:
            s3 = boto3.client("s3", config=Config(signature_version=UNSIGNED))  # anonymous client

        s3.download_file(config["bucket"], config["key"], "/tmp/model.pkl")
        self.model = pickle.load(open("/tmp/model.pkl", "rb"))

    # TODO update mapping function, this fails if request_id is not taken
    def predict(self, payload, request_id):
        measurements = [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]

        label_id = self.model.predict([measurements])[0]

        # TODO if this returns a string, it throws the following error:
        # error: json: cannot unmarshal string into Go value of type map[string]interface {}
        return {"result": labels[label_id]}
