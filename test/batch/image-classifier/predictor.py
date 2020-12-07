# WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.23.*, run `git checkout -b 0.23` or switch to the `0.23` branch on GitHub)

import os
import requests
from PIL import Image
from io import BytesIO
import json
import re

# labels  "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
# bucket, key


class PythonPredictor:
    def __init__(self, config, job_spec):
        import re
        import boto3
        from torchvision import transforms
        import torchvision

        self.model = torchvision.models.alexnet(pretrained=True).eval()

        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )

        self.labels = requests.get(config["labels"]).text.split("\n")[1:]

        self.s3 = boto3.client("s3")  # initialize S3 client to save results

        self.bucket, self.key = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
        self.key = os.path.join(self.key, job_spec["job_id"])

    def predict(self, payload, batch_id):
        import json
        from PIL import Image
        import torch

        tensor_list = []
        for image_url in payload:  # download and preprocess each image
            img_pil = Image.open(BytesIO(requests.get(image_url).content))
            tensor_list.append(self.preprocess(img_pil))

        img_tensor = torch.stack(tensor_list)
        with torch.no_grad():  # classify the batch of images
            prediction = self.model(img_tensor)
        _, indices = prediction.max(1)

        results = [  # extract predicted classes
            {"url": payload[i], "class": self.labels[class_idx]}
            for i, class_idx in enumerate(indices)
        ]

        # save results
        self.s3.put_object(
            Bucket=self.bucket, Key=f"{self.key}/{batch_id}.json", Body=json.dumps(results)
        )
