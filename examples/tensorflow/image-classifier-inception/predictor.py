# WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.22.*, run `git checkout -b 0.22` or switch to the `0.22` branch on GitHub)

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
        image = requests.get(payload["url"]).content
        decoded_image = np.asarray(Image.open(BytesIO(image)), dtype=np.float32) / 255
        model_input = {"images": np.expand_dims(decoded_image, axis=0)}
        prediction = self.client.predict(model_input)
        return self.labels[np.argmax(prediction["classes"])]
