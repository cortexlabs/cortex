from io import BytesIO

import requests
import torch
from PIL import Image
from torchvision import models
from torchvision import transforms


class PythonPredictor:
    def __init__(self, config):
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"using device: {self.device}")

        model = models.detection.fasterrcnn_resnet50_fpn(pretrained=True).to(self.device)
        model.eval()

        self.preprocess = transforms.Compose([transforms.ToTensor()])

        with open("/mnt/project/coco_labels.txt") as f:
            self.coco_labels = f.read().splitlines()

        self.model = model

    def predict(self, payload):
        threshold = float(payload["threshold"])
        image = requests.get(payload["url"]).content
        img_pil = Image.open(BytesIO(image))
        img_tensor = self.preprocess(img_pil).to(self.device)
        img_tensor.unsqueeze_(0)

        with torch.no_grad():
            pred = self.model(img_tensor)

        predicted_class = [self.coco_labels[i] for i in pred[0]["labels"].cpu().tolist()]
        predicted_boxes = [
            [(i[0], i[1]), (i[2], i[3])] for i in pred[0]["boxes"].detach().cpu().tolist()
        ]
        predicted_score = pred[0]["scores"].detach().cpu().tolist()
        predicted_t = [predicted_score.index(x) for x in predicted_score if x > threshold]
        if len(predicted_t) == 0:
            return [], []

        predicted_t = predicted_t[-1]
        predicted_boxes = predicted_boxes[: predicted_t + 1]
        predicted_class = predicted_class[: predicted_t + 1]
        return predicted_boxes, predicted_class
