# this is an example for cortex release 0.16 and may not deploy correctly on other releases of cortex

import boto3
from botocore import UNSIGNED
from botocore.client import Config
from encoder import get_encoder


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client
        s3_client = boto3.client("s3", config=Config(signature_version=UNSIGNED))
        self.encoder = get_encoder(s3_client)

    def predict(self, payload):
        model_input = {"context": [self.encoder.encode(payload["text"])]}
        prediction = self.client.predict(model_input)
        return self.encoder.decode(prediction["sample"])
