import boto3
from botocore import UNSIGNED
from botocore.client import Config
import mlflow.sklearn
import numpy as np
import re
import os


class PythonPredictor:
    def __init__(self, config):
        model_path = "/tmp/model"
        os.makedirs(model_path, exist_ok=True)

        # download mlflow model folder from S3
        bucket, prefix = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        response = s3.list_objects_v2(Bucket=bucket, Prefix=prefix)
        for s3_obj in response["Contents"]:
            obj_key = s3_obj["Key"]
            s3.download_file(bucket, obj_key, os.path.join(model_path, os.path.basename(obj_key)))

        self.model = mlflow.sklearn.load_model(model_path)

    def predict(self, payload):
        model_input = [
            payload["cylinders"],
            payload["displacement"],
            payload["horsepower"],
            payload["weight"],
            payload["acceleration"],
        ]

        result = self.model.predict([model_input])
        return np.asscalar(result)
