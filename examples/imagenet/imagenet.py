import requests
import numpy as np
import base64
from PIL import Image
from io import BytesIO
import math
from cortex.lib.log import get_logger

logger = get_logger()

labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")


def pre_inference(sample, metadata):
    decoded = base64.b64decode(sample["image"])
    decoded_image = np.asarray(Image.open(BytesIO(decoded)), dtype=np.float32) / 255
    logger.info(decoded_image)
    return {"images": [decoded_image.tolist()]}


def post_inference(prediction, metadata):
    classes = prediction["response"]["classes"]
    return {"class": labels[np.argmax(classes)]}
