import re
import torch
from model import IrisNet


model = IrisNet()


def init(model_path, metadata):
    model.load_state_dict(torch.load(model_path))
    model.eval()


labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def predict(payload, metadata):
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

    output = model(input_tensor)
    return labels[torch.argmax(output[0])]
