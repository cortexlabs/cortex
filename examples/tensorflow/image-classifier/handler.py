import requests
import numpy as np
import base64
from PIL import Image
from io import BytesIO

labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")


def pre_inference(sample, signature, metadata):
    if "url" in sample:
        image = requests.get(sample["url"]).content
    elif "base64" in sample:
        image = base64.b64decode(sample["base64"])

    decoded_image = np.asarray(Image.open(BytesIO(image)), dtype=np.float32) / 255
    return {"images": np.expand_dims(decoded_image, axis=0)}


def post_inference(prediction, signature, metadata):
    return labels[np.argmax(prediction["classes"])]
