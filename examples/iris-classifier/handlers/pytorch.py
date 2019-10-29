import numpy as np
import os
import torch

from model import Net

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def model_init(ctx, api):
    _, prefix = ctx.storage.deconstruct_s3_path(api["model"])
    model_path = os.path.join("/mnt/model", os.path.basename(prefix))
    model = Net()
    model.load_state_dict(torch.load(model_path))
    return model


def pre_inference(sample, signature, metadata):
    return torch.FloatTensor(
        [
            [
                sample["sepal_length"],
                sample["sepal_width"],
                sample["petal_length"],
                sample["petal_width"],
            ]
        ]
    )


def post_inference(prediction, signature, metadata):
    _, index = prediction.max(1)
    return labels[index]


# serving.py?
# only accept pickles?
