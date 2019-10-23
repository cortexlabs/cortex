import numpy as np
import os
import torch
import dill

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def model_init(ctx, api):
    _, prefix = ctx.storage.deconstruct_s3_path(api["model"])
    model_path = os.path.join("/mnt/model", os.path.basename(prefix))
    model = torch.load(model_path, pickle_module=dill)
    return model


def pre_inference(sample, metadata):
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


def post_inference(prediction, metadata):
    _, index = prediction.max(1)
    return labels[index]


# serving.py?
# only accept pickles?
