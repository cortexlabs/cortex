# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import requests
import numpy as np
from PIL import Image
from io import BytesIO

labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")


def pre_inference(payload, signature, metadata):
    image = requests.get(payload["url"]).content
    decoded_image = np.asarray(Image.open(BytesIO(image)), dtype=np.float32) / 255
    return {"images": np.expand_dims(decoded_image, axis=0)}


def post_inference(prediction, signature, metadata):
    return labels[np.argmax(prediction["classes"])]
