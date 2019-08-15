import requests
import numpy as np

labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
)


def pre_inference(sample, metadata):
    return {"images": [np.zeros((299, 299, 3)).tolist()]}


def post_inference(prediction, metadata):
    return {"sentiment": labels[prediction["response"]["labels"][0]]}
