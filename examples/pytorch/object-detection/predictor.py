# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

from io import BytesIO

import requests
import torch
from PIL import Image
from torchvision import models
from torchvision import transforms


class PythonPredictor:
    def __init__(self, config):
        model = models.detection.fasterrcnn_resnet50_fpn(pretrained=True)
        model.eval()

        self.preprocess = transforms.Compose([transforms.ToTensor()])

        with open("/mnt/project/coco_labels.txt") as f:
            self.coco_labels = f.read().splitlines()

        self.model = model

    def predict(self, payload):
        threshold = float(payload["threshold"])
        image = requests.get(payload["url"]).content
        img_pil = Image.open(BytesIO(image))

        img_tensor = self.preprocess(img_pil)

        img_tensor.unsqueeze_(0)

        with torch.no_grad():
            pred = self.model(img_tensor)

        pred_class = [self.coco_labels[i] for i in list(pred[0]["labels"].numpy())]
        pred_boxes = [[(i[0], i[1]), (i[2], i[3])] for i in list(pred[0]["boxes"].detach().numpy())]
        pred_score = list(pred[0]["scores"].detach().numpy())
        pred_t = [pred_score.index(x) for x in pred_score if x > threshold]
        if len(pred_t) == 0:
            return [], []

        pred_t = pred_t[-1]
        pred_boxes = pred_boxes[: pred_t + 1]
        pred_class = pred_class[: pred_t + 1]
        return pred_boxes, pred_class
