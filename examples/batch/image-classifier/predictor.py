# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import os
import requests
import torch
import torchvision
from torchvision import transforms
from PIL import Image
from io import BytesIO
import boto3
from botocore import UNSIGNED
from botocore.client import Config
import json
import os


class PythonPredictor:
    def __init__(self, config, job_spec):
        # device = "cuda" if torch.cuda.is_available() else "cpu"
        # print(f"using device: {device}")

        # model = torchvision.models.alexnet(pretrained=True).to(device)
        # model.eval()
        # # https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
        # normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])

        # self.preprocess = transforms.Compose(
        #     [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        # )
        # self.labels = requests.get(
        #     "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        # ).text.split("\n")[1:]
        # self.model = model
        # self.device = device
        # self.job_id = job_spec["job_id"]
        self.counter = 0
        pass

    def predict(self, payload, batch_id):
        print(f"received payload of length {len(payload)}")
        self.counter += len(payload)
        # tensor_list = []

        # for image_url in payload:
        #     image = requests.get(image_url).content
        #     img_pil = Image.open(BytesIO(image))
        #     tensor_list.append(self.preprocess(img_pil))
        #     img_tensor = torch.stack(tensor_list)
        #     img_tensor = img_tensor.to("cpu")
        #     with torch.no_grad():
        #         prediction = self.model(img_tensor)
        #     _, indices = prediction.max(1)

        # results = []
        # for index in indices:
        #     results.append(self.labels[index])

        # # you can write the output to storage yourself and NOT return anything
        # # if you return a value, the value must be bytes, string or a JSON parseable object
        # return results  # uploaded to s3://<CORTEX_BUCKET/job_results/<API_NAME>/<JOB_ID>/<BATCH_ID>

    def on_job_complete(self):
        print(self.counter)
        time.sleep(20)
        print("completed on_job_complete")
        # raise Exception("on_job_complete")
