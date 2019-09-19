import requests
import numpy as np
import base64
from PIL import Image
from io import BytesIO
from torchvision import transforms

labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")[1:]


# https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
preprocess = transforms.Compose(
    [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
)


def pre_inference(sample, metadata):
    if "url" in sample:
        image = requests.get(sample["url"]).content
    elif "base64" in sample:
        image = base64.b64decode(sample["base64"])

    img_pil = Image.open(BytesIO(image))
    img_tensor = preprocess(img_pil)
    img_tensor.unsqueeze_(0)
    return img_tensor.numpy()


def post_inference(prediction, metadata):
    return labels[np.argmax(np.array(prediction).squeeze())]
