import torch
from src.model import Net
import boto3

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

model = Net()


def init(metadata):
    print(metadata)
    s3 = boto3.client("s3", region_name=metadata["region"])
    s3.download_file(metadata["bucket"], metadata["key"], "iris_model.pth")
    model.load_state_dict(torch.load("iris_model.pth"))
    model.eval()


def inference(sample, metadata):
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

