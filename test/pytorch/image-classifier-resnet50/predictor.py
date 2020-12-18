import os
import torch
import cv2
import numpy as np
import requests
import re
import boto3
from botocore import UNSIGNED
from botocore.client import Config
from torchvision import models, transforms, datasets


def get_url_image(url_image):
    """
    Get numpy image from URL image.
    """
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_COLOR)
    image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
    return image


class PythonPredictor:
    def __init__(self, config):
        # load classes
        classes = requests.get(config["classes"]).json()
        self.idx2label = [classes[str(k)][1] for k in range(len(classes))]

        # create s3 client
        if os.environ.get("AWS_ACCESS_KEY_ID"):
            s3 = boto3.client("s3")  # client will use your credentials if available
        else:
            s3 = boto3.client("s3", config=Config(signature_version=UNSIGNED))  # anonymous client

        # download the model
        model_path = config["model_path"]
        model_name = config["model_name"]
        bucket, key = re.match("s3://(.+?)/(.+)", model_path).groups()
        s3.download_file(bucket, os.path.join(key, model_name), model_name)

        # load the model
        self.device = None
        if config["device"] == "gpu":
            self.device = torch.device("cuda")
            self.model = models.resnet50()
            self.model.load_state_dict(torch.load(model_name, map_location="cuda:0"))
            self.model.eval()
            self.model = self.model.to(self.device)
        elif config["device"] == "cpu":
            self.model = models.resnet50()
            self.model.load_state_dict(torch.load(model_name))
            self.model.eval()
        elif config["device"] == "inf":
            import torch_neuron

            self.model = torch.jit.load(model_name)
        else:
            raise RuntimeError("invalid predictor: config: must be cpu, gpu, or inf")

        # save normalization transform for later use
        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        self.transform = transforms.Compose(
            [
                transforms.ToPILImage(),
                transforms.Resize(config["input_shape"]),
                transforms.ToTensor(),
                normalize,
            ]
        )

    def predict(self, payload):
        # preprocess image
        image = get_url_image(payload["url"])
        image = self.transform(image)
        image = torch.tensor(image.numpy()[np.newaxis, ...])

        # predict
        if self.device:
            results = self.model(image.to(self.device))
        else:
            results = self.model(image)

        # Get the top 5 results
        top5_idx = results[0].sort()[1][-5:]

        # Lookup and print the top 5 labels
        top5_labels = [self.idx2label[idx] for idx in top5_idx]
        top5_labels = top5_labels[::-1]

        return top5_labels
