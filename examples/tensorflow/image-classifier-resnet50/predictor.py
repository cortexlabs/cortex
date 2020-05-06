import os
import cv2
import numpy as np
import requests
import json


def get_url_image(url_image):
    """
    Get numpy image from URL image.
    """
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_COLOR)
    image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
    return image


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

        # load classes
        classes = requests.get(config["classes"]).json()
        self.idx2label = [classes[str(k)][1] for k in range(len(classes))]

        self.input_shape = tuple(config["input_shape"])

    def predict(self, payload):
        # preprocess image
        image = get_url_image(payload["url"])
        image = cv2.resize(image, self.input_shape, interpolation=cv2.INTER_NEAREST)
        image = {"input": image[np.newaxis, ...]}

        # predict
        results = self.client.predict(image)["output"]
        results = np.argsort(results)

        # Lookup and print the top 5 labels
        top5_idx = results[-5:]
        top5_labels = [self.idx2label[idx] for idx in top5_idx]
        top5_labels = top5_labels[::-1]

        return top5_labels
