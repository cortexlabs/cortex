import re
import torch
import os
import boto3
from botocore import UNSIGNED
from botocore.client import Config
from model import IrisNet
import time

labels = ["setosa", "versicolor", "virginica"]


class Handler:
    def __init__(self, config):
        # download the model
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "/tmp/model.pth")

        # initialize the model
        model = IrisNet()
        model.load_state_dict(torch.load("/tmp/model.pth"))
        model.eval()

        self.model = model

    def handle_post(self, payload):
        # Convert the request to a tensor and pass it into the model
        input_tensor = torch.FloatTensor(
            [
                [
                    payload["sepal_length"],
                    payload["sepal_width"],
                    payload["petal_length"],
                    payload["petal_width"],
                ]
            ]
        )

        # Run the prediction
        output = self.model(input_tensor)

        # Translate the model output to the corresponding label string
        return labels[torch.argmax(output[0])]

    def handle_get(self):
        return "handle_get response"

    def handle_put(self):
        def _callback():
            print("preparing callback")
            time.sleep(3.0)
            print("callback called after 3 seconds")

        return "handle_put response", _callback
