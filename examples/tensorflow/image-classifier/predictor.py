# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import requests
import numpy as np
from PIL import Image
from io import BytesIO

labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

    def predict(self, payload):
        image = requests.get(payload["url"]).content
        decoded_image = np.asarray(Image.open(BytesIO(image)), dtype=np.float32) / 255
        model_input = {"images": np.expand_dims(decoded_image, axis=0)}
        prediction = self.client.predict(model_input)
        return labels[np.argmax(prediction["classes"])]
