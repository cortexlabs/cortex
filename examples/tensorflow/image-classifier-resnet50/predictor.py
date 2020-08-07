# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import os
import cv2
import numpy as np
import requests
import json
import base64


def get_url_image(url_image):
    """
    Get numpy image from URL image.
    """
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_COLOR)
    image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
    return image


def decode_image(image):
    """
    Decode the image from the payload.
    """
    img = base64.b64decode(image)
    jpg_as_np = np.frombuffer(img, dtype=np.uint8)
    img = cv2.imdecode(jpg_as_np, flags=cv2.IMREAD_COLOR)
    return img


def prepare_image(image, input_shape, input_key):
    """
    Prepares an image for the TFS client.
    """
    img = cv2.resize(image, input_shape, interpolation=cv2.INTER_NEAREST)
    img = {input_key: img[np.newaxis, ...]}
    return img


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

        # load classes
        classes = requests.get(config["classes"]).json()
        self.idx2label = [classes[str(k)][1] for k in range(len(classes))]

        self.input_shape = tuple(config["input_shape"])
        self.input_key = str(config["input_key"])
        self.output_key = str(config["output_key"])

    def predict(self, payload):
        # preprocess image
        payload_keys = payload.keys()
        if "img" in payload_keys:
            img = payload["img"]
            img = decode_image(img)
        elif "url" in payload_keys:
            img = get_url_image(payload["url"])
        else:
            return None
        img = prepare_image(img, self.input_shape, self.input_key)

        # predict
        results = self.client.predict(img)[self.output_key]
        results = np.argsort(results)

        # Lookup and print the top 5 labels
        top5_idx = results[-5:]
        top5_labels = [self.idx2label[idx] for idx in top5_idx]
        top5_labels = top5_labels[::-1]

        return top5_labels
