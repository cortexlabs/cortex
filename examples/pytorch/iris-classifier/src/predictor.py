import re
import torch
from model import IrisNet

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

model = IrisNet()


def init(model_path, metadata):
    model.load_state_dict(torch.load(model_path))
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
