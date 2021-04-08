import os
import boto3
from botocore import UNSIGNED
from botocore.client import Config
from encoder import get_encoder


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client
        s3 = boto3.client("s3")
        self.encoder = get_encoder(s3)

    def predict(self, payload):
        model_input = {"context": [self.encoder.encode(payload["text"])]}
        prediction = self.client.predict(model_input)
        return self.encoder.decode(prediction["sample"])
