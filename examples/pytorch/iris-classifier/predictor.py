# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import re
import torch
import os
import boto3
from botocore import UNSIGNED
from botocore.client import Config
from model import IrisNet
import time
import uuid

labels = ["setosa", "versicolor", "virginica"]

import logging

logger = logging.getLogger(__name__)

uuidStr = str(uuid.uuid4())


class PythonPredictor:
    def __init__(self, config):
        print(config)
        # download the model
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()

        if os.environ.get("AWS_ACCESS_KEY_ID"):
            s3 = boto3.client("s3")  # client will use your credentials if available
        else:
            s3 = boto3.client("s3", config=Config(signature_version=UNSIGNED))  # anonymous client

        s3.download_file(bucket, key, "/tmp/model.pth")

        # initialize the model
        model = IrisNet()
        model.load_state_dict(torch.load("/tmp/model.pth"))
        model.eval()

        self.config = config
        self.model = model

    def predict(self, payload):
        # Convert the request to a tensor and pass it into the model
        time.sleep(60)
        print(payload)

        # input_tensor = torch.FloatTensor(
        #     [
        #         [
        #             payload["sepal_length"],
        #             payload["sepal_width"],
        #             payload["petal_length"],
        #             payload["petal_width"],
        #         ]
        #     ]
        # )

        # # Run the prediction
        # output = self.model(input_tensor)

        # # Translate the model output to the corresponding label string
        # return labels[torch.argmax(output[0])]
