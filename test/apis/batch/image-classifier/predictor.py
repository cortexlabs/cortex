import os
import requests
import torch
import torchvision
from torchvision import transforms
from PIL import Image
from io import BytesIO
import boto3
import json
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

        if len(config.get("dest_s3_dir", "")) == 0:
            raise Exception("'dest_s3_dir' field was not provided in job submission")

        self.s3 = boto3.client("s3")

        self.bucket, self.key = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
        self.key = os.path.join(self.key, job_spec["job_id"])

    def predict(self, payload, batch_id):
        tensor_list = []

        # download and preprocess each image
        for image_url in payload:
            if image_url.startswith("s3://"):
                bucket, image_key = re.match("s3://(.+?)/(.+)", image_url).groups()
                image_bytes = self.s3.get_object(Bucket=bucket, Key=image_key)["Body"].read()
            else:
                image_bytes = requests.get(image_url).content

            img_pil = Image.open(BytesIO(image_bytes))
            tensor_list.append(self.preprocess(img_pil))

        # classify the batch of images
        img_tensor = torch.stack(tensor_list)
        with torch.no_grad():
            prediction = self.model(img_tensor)
        _, indices = prediction.max(1)

        # extract predicted classes
        results = [
            {"url": payload[i], "class": self.labels[class_idx]}
            for i, class_idx in enumerate(indices)
        ]
        json_output = json.dumps(results)

        # save results
        self.s3.put_object(Bucket=self.bucket, Key=f"{self.key}/{batch_id}.json", Body=json_output)

    def on_job_complete(self):
        all_results = []

        # aggregate all classifications
        paginator = self.s3.get_paginator("list_objects_v2")
        for page in paginator.paginate(Bucket=self.bucket, Prefix=self.key):
            for obj in page["Contents"]:
                body = self.s3.get_object(Bucket=self.bucket, Key=obj["Key"])["Body"]
                all_results += json.loads(body.read().decode("utf8"))

        # save single file containing aggregated classifications
        self.s3.put_object(
            Bucket=self.bucket,
            Key=os.path.join(self.key, "aggregated_results.json"),
            Body=json.dumps(all_results),
        )
