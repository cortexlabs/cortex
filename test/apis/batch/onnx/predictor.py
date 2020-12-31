import requests
import numpy as np
import base64
from PIL import Image
from io import BytesIO
from torchvision import transforms
import boto3
import json
import re
import os


class ONNXPredictor:
    def __init__(self, onnx_client, config, job_spec):
        self.client = onnx_client

        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")[1:]

        # https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )

        if len(config.get("dest_s3_dir", "")) == 0:
            raise Exception("'dest_s3_dir' field was not provided in job submission")

        self.s3 = boto3.client("s3")

        self.bucket, self.key = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
        self.key = os.path.join(self.key, job_spec["job_id"])

    def predict(self, payload, batch_id):
        arr_list = []

        # download and preprocess each image
        for image_url in payload:
            if image_url.startswith("s3://"):
                bucket, image_key = re.match("s3://(.+?)/(.+)", image_url).groups()
                image_bytes = self.s3.get_object(Bucket=bucket, Key=image_key)["Body"].read()
            else:
                image_bytes = requests.get(image_url).content

            img_pil = Image.open(BytesIO(image_bytes))
            arr_list.append(self.preprocess(img_pil).numpy())

        # classify the batch of images
        imgs_arr = np.stack(arr_list, axis=0)
        result = self.client.predict(imgs_arr)

        # extract predicted classes
        predicted_classes = np.argmax(result[0], axis=1)
        results = [
            {"url": payload[i], "class": self.labels[class_idx]}
            for i, class_idx in enumerate(predicted_classes)
        ]

        # save results
        json_output = json.dumps(results)
        self.s3.put_object(Bucket=self.bucket, Key=f"{self.key}/{batch_id}.json", Body=json_output)
