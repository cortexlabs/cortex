# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import re
import torch
import boto3
from model import IrisNet

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config):
        # download the model
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "model.pth")

        # initialize the model
        model = IrisNet()
        model.load_state_dict(torch.load("model.pth"))
        model.eval()

        self.model = model

    def predict(self, payload):
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
