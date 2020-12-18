import os
import cv2
import numpy as np
import requests
import imageio
import json
import base64


def read_image(payload):
    """
    Read JPG image from {"url": "https://..."} or from a bytes object.
    """
    if isinstance(payload, bytes):
        jpg_as_np = np.frombuffer(payload, dtype=np.uint8)
        img = cv2.imdecode(jpg_as_np, flags=cv2.IMREAD_COLOR)
    elif isinstance(payload, dict) and "url" in payload.keys():
        img = imageio.imread(payload["url"])
    else:
        return None
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
        img = read_image(payload)
        if img is None:
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
