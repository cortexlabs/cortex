# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import os
import requests
import torch
import torchvision
from torchvision import transforms
from PIL import Image
from io import BytesIO
import boto3
import json
import os
import re


class PythonPredictor:
    def __init__(self, config, job_spec):
        self.model = torchvision.models.alexnet(pretrained=True).eval()

        # https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )

        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")[1:]

    def predict(self, payload, batch_id):
        tensor_list = []

        for image_url in payload:
            bucket, key = re.match("s3://(.+?)/(.+)", image_url).groups()
            file_byte_string = self.s3.get_object(Bucket=bucket, Key=key)["Body"].read()
            img_pil = Image.open(BytesIO(file_byte_string))
            tensor_list.append(self.preprocess(img_pil))

        img_tensor = torch.stack(tensor_list)

        with torch.no_grad():
            prediction = self.model(img_tensor)
        _, indices = prediction.max(1)

        results = []
        for index in indices:
            results.append(self.labels[index])

        self.s3.put_object(
            Bucket=self.bucket,
            Key=os.path.join(self.key, batch_id + ".json"),
            Body=str(json.dumps(results)),
        )

    def on_job_complete(self):
        all_results = []

        # download all of the results
        for obj in self.s3.list_objects_v2(Bucket=self.bucket, Prefix=self.key).Contents:
            if obj["Size"] > 0:
                body = self.s3.get_object(Bucket=self.bucket, Prefix=self.key)["Body"]
                all_results.append(json.loads(body.read().decode("utf8")))

        newline_delimited_json = "\n".join(json.dumps(result) for result in all_results)
        self.s3.put_object(
            Bucket=self.bucket,
            Key=os.path.join(self.key, "aggregated_results" + ".csv"),
            Body=str(newline_delimited_json),
        )
