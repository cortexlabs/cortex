import torch
from my_model import IrisNet
import boto3
import re

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

model = IrisNet()


def init(metadata):
    s3 = boto3.client("s3")
    bucket, key = re.match(r"s3:\/\/(.+?)\/(.+)", metadata["model"]).groups()
    s3.download_file(bucket, key, "iris_model.pth")
    model.load_state_dict(torch.load("iris_model.pth"))
    model.eval()


def predict(sample, metadata):
    input_tensor = torch.FloatTensor(
        [
            [
                sample["sepal_length"],
                sample["sepal_width"],
                sample["petal_length"],
                sample["petal_width"],
            ]
        ]
    )

    output = model(input_tensor)
    return labels[torch.argmax(output[0])]
