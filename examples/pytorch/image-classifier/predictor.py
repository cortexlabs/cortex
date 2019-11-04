import requests
from PIL import Image
from io import BytesIO
from torchvision import transforms
import torchvision
import torch

model = torchvision.models.alexnet(pretrained=True)
model.eval()

# https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
preprocess = transforms.Compose(
    [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
)

labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")[1:]


def predict(sample, metadata):
    image = requests.get(sample["url"]).content
    img_pil = Image.open(BytesIO(image))
    img_tensor = preprocess(img_pil)
    img_tensor.unsqueeze_(0)
    with torch.no_grad():
        prediction = model(img_tensor)
    _, index = prediction[0].max(0)
    return labels[index]
