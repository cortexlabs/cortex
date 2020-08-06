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

        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )

        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")[1:]

        self.s3 = boto3.client("s3")
        self.bucket, self.key = re.match("s3://(.+?)/(.+)", config["s3_dir"]).groups()
        self.key = os.path.join(self.key, job_spec["job_id"])

    def predict(self, payload, batch_id):
        tensor_list = []

        # download and preprocess each image
        for image_url in payload:
            image = requests.get(image_url).content
            img_pil = Image.open(BytesIO(image))
            tensor_list.append(self.preprocess(img_pil))

        # classify the batch of images
        img_tensor = torch.stack(tensor_list)
        with torch.no_grad():
            prediction = self.model(img_tensor)
        _, indices = prediction.max(1)

        results = [self.labels[index] for index in indices]
        json_output = json.dumps(results)

        self.s3.put_object(Bucket=self.bucket, Key=f"{self.key}/{batch_id}.json", Body=json_output)

    def on_job_complete(self):
        all_results = []

        # download all of the results
        for obj in self.s3.list_objects_v2(Bucket=self.bucket, Prefix=self.key)["Contents"]:
            if obj["Size"] > 0:
                print(obj)
                body = self.s3.get_object(Bucket=self.bucket, Key=obj["Key"])["Body"]
                all_results.append(json.loads(body.read().decode("utf8")))

        newline_delimited_results = "\n".join(json.dumps(result) for result in all_results)
        self.s3.put_object(
            Bucket=self.bucket,
            Key=os.path.join(self.key, "aggregated_results.csv"),
            Body=str(newline_delimited_results),
        )
