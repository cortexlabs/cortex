# this is an example for cortex release 0.16 and may not deploy correctly on other releases of cortex

import os
import re
import boto3
from botocore import UNSIGNED
from botocore.client import Config
from fastai.text import load_learner


class PythonPredictor:
    def __init__(self, config):
        # download the model
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3", config=Config(signature_version=UNSIGNED))
        os.mkdir("/tmp/model")
        s3.download_file(bucket, key, "/tmp/model/export.pkl")

        self.predictor = load_learner("/tmp/model")

    def predict(self, payload):
        prediction = self.predictor.predict(payload["text"])
        return prediction[0].obj
