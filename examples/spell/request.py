import numpy as np
import requests
from PIL import Image

TF_SERVING_API = (
    "https://acea1a9abbdca11e9b29c0ea6849e7fb-276107737.us-west-2.elb.amazonaws.com/spell/spell"
)


def load_image(path):
    img = Image.open(path)
    img.load()
    data = np.asarray(img, dtype="uint8")
    data = np.delete(data, 0, 2)
    data = np.expand_dims(data, axis=0)
    return data


def load_zeros():
    return np.zeros((1, 1024, 768), dtype="uint8")


# image = load_image("./mower.jpg")
# image = load_image("./mower2.jpg")
image = load_zeros()

print(image.shape, image.dtype)

predict_body = {"samples": [{"images": [image.tolist()]}]}

print("starting request")
result = requests.post(TF_SERVING_API, json=predict_body, verify=False)
print("finished request")

print(result.text)
