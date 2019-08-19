import requests
import numpy as np
import base64
from PIL import Image
from io import BytesIO
import math


labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")


def pre_inference(sample, metadata):
    response = requests.get(sample["link"])
    decoded_image = np.asarray(Image.open(BytesIO(response.content)), dtype=np.float32) / 255
    return {"images": [decoded_image.tolist()]}


def post_inference(prediction, metadata):
    classes = prediction["response"]["classes"]
    return {"class": labels[np.argmax(classes)]}
