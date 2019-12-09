import re
import torch
import boto3
from model import IrisNet


class Predictor:
    def __init__(self, config):
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "model.pth")

        model = IrisNet()
        model.load_state_dict(torch.load("model.pth"))
        model.eval()

        self.model = model
        self.labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

    def predict(self, payload):
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

        output = self.model(input_tensor)
        return self.labels[torch.argmax(output[0])]
