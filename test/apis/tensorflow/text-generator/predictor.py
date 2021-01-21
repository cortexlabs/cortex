import os
import boto3
from botocore import UNSIGNED
from botocore.client import Config
from encoder import get_encoder


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

        if os.environ.get("AWS_ACCESS_KEY_ID"):
            s3 = boto3.client("s3")  # client will use your credentials if available
        else:
            s3 = boto3.client("s3", config=Config(signature_version=UNSIGNED))  # anonymous client

        self.encoder = get_encoder(s3)

    def predict(self, payload):
        model_input = {"context": [self.encoder.encode(payload["text"])]}
        prediction = self.client.predict(model_input)
        return self.encoder.decode(prediction["sample"])
