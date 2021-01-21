# BatchAPI

Deploy batch APIs that can orchestrate distributed batch inference jobs on large datasets.

## Key features

* Distributed inference
* Automatic batch retries
* Collect failed batches for debugging
* Metrics and log aggregation
* `on_job_complete` webhook
* Scale to 0

## How it works

### Install cortex

```bash
$ pip install cortex
```

### Spin up a cluster on AWS

```bash
$ cortex cluster up
```

### Define a batch API

```python
# batch.py

import cortex

class PythonPredictor:
    def __init__(self, config, job_spec):
        from torchvision import transforms
        import torchvision
        import requests
        import boto3
        import re

        self.model = torchvision.models.alexnet(pretrained=True).eval()
        self.labels = requests.get(config["labels"]).text.split("\n")[1:]

        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )

        self.s3 = boto3.client("s3")  # initialize S3 client to save results
        self.bucket, self.key = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
        self.key = os.path.join(self.key, job_spec["job_id"])

    def predict(self, payload, batch_id):
        import json
        import torch
        from PIL import Image
        from io import BytesIO
        import requests

        tensor_list = []
        for image_url in payload:  # download and preprocess each image
            img_pil = Image.open(BytesIO(requests.get(image_url).content))
            tensor_list.append(self.preprocess(img_pil))

        img_tensor = torch.stack(tensor_list)
        with torch.no_grad():  # classify the batch of images
            prediction = self.model(img_tensor)
        _, indices = prediction.max(1)

        results = [{"url": payload[i], "class": self.labels[class_idx]} for i, class_idx in enumerate(indices)]
        self.s3.put_object(Bucket=self.bucket, Key=f"{self.key}/{batch_id}.json", Body=json.dumps(results))

requirements = ["torch", "boto3", "pillow", "torchvision", "requests"]

api_spec = {
    "name": "image-classifier",
    "kind": "BatchAPI",
    "predictor": {
        "config": {
            "labels": "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        }
    }
}

cx = cortex.client("aws")
cx.create_api(api_spec, predictor=PythonPredictor, requirements=requirements)
```

### Deploy to your Cortex cluster on AWS

```bash
$ python batch.py
```

### Describe the Batch API

```bash
$ cortex get image-classifier
```

### Submit a job

```python
import cortex
import requests

cx = cortex.client("aws")
batch_endpoint = cx.get_api("image-classifier")["endpoint"]

dest_s3_dir = # specify S3 directory for the results (make sure your cluster has access to this bucket)

job_spec = {
    "workers": 1,
    "item_list": {
        "items": [
            "https://i.imgur.com/PzXprwl.jpg",
            "https://i.imgur.com/E4cOSLw.jpg",
            "https://user-images.githubusercontent.com/4365343/96516272-d40aa980-1234-11eb-949d-8e7e739b8345.jpg",
            "https://i.imgur.com/jDimNTZ.jpg",
            "https://i.imgur.com/WqeovVj.jpg"
        ],
        "batch_size": 2
    },
    "config": {
        "dest_s3_dir": dest_s3_dir
    }
}

response = requests.post(batch_endpoint, json=job_spec)

print(response.text)
# > {"job_id":"69b183ed6bdf3e9b","api_name":"image-classifier", "config": {"dest_s3_dir": ...}}
```

### Monitor the job

```bash
$ cortex get image-classifier 69b183ed6bdf3e9b
```

### Stream job logs

```bash
$ cortex logs image-classifier 69b183ed6bdf3e9b
```

### View the results

Once the job is complete, you should be able to find the results of the batch job in the S3 directory you've specified.

### Delete the Batch API

```bash
$ cortex delete image-classifier
```
