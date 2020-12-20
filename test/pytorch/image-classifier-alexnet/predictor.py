import requests
import torch
import torchvision
from torchvision import transforms
from PIL import Image
from io import BytesIO


class PythonPredictor:
    def __init__(self, config):
        device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"using device: {device}")

        model = torchvision.models.alexnet(pretrained=True).to(device)
        model.eval()
        # https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])

        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )
        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")[1:]
        self.model = model
        self.device = device

    def predict(self, payload):
        image = requests.get(payload["url"]).content
        img_pil = Image.open(BytesIO(image))
        img_tensor = self.preprocess(img_pil)
        img_tensor.unsqueeze_(0)
        img_tensor = img_tensor.to(self.device)
        with torch.no_grad():
            prediction = self.model(img_tensor)
        _, index = prediction[0].max(0)
        return self.labels[index]
