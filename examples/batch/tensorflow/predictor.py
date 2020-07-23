# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import requests
import numpy as np
from PIL import Image
from io import BytesIO


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client
        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")

    def predict(self, payload):
        image_list = []

        for image_url in payload:
            image = requests.get(image_url).content
            decoded_image = np.asarray(Image.open(BytesIO(image)), dtype=np.float32) / 255
            image_list.append(decoded_image)

        model_input = {"images": np.stack(image_list)}
        prediction = self.client.predict(model_input)
        return self.labels[np.argmax(prediction["classes"])]
