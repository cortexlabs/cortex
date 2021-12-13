import os, json, re
import requests
import torch
import torchvision
import boto3
import uuid

from typing import List
from PIL import Image
from io import BytesIO
from torchvision import transforms
from fastapi import FastAPI, Request, status
from fastapi.responses import PlainTextResponse

app = FastAPI()
app.device = os.getenv("TARGET_DEVICE", "cpu")
app.ready = False
s3 = boto3.client("s3")


@app.on_event("startup")
def startup():
    # read job spec
    with open("/cortex/spec/job.json", "r") as f:
        job_spec = json.load(f)
    print(json.dumps(job_spec, indent=2))

    # get metadata
    config = job_spec["config"]
    app.job_id = job_spec["job_id"]
    if len(config.get("dest_s3_dir", "")) == 0:
        raise Exception("'dest_s3_dir' field was not provided in job submission")

    # s3 info
    app.bucket, app.key = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
    app.key = os.path.join(app.key, app.job_id)

    # loading model
    normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
    app.preprocess = transforms.Compose(
        [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
    )
    app.labels = requests.get(
        "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
    ).text.split("\n")[1:]
    app.model = torchvision.models.alexnet(pretrained=True).eval().to(app.device)

    app.ready = True


@app.get("/healthz")
def healthz():
    if app.ready:
        return PlainTextResponse("ok")
    return PlainTextResponse("service unavailable", status_code=status.HTTP_503_SERVICE_UNAVAILABLE)


@app.post("/")
def handle_batch(image_urls: List[str]):
    tensor_list = []

    # download and preprocess each image
    for image_url in image_urls:
        if image_url.startswith("s3://"):
            bucket, image_key = re.match("s3://(.+?)/(.+)", image_url).groups()
            image_bytes = s3.get_object(Bucket=bucket, Key=image_key)["Body"].read()
        elif image_url.startswith("http"):
            image_bytes = requests.get(image_url).content
        else:
            raise RuntimeError(f"{image_url}: invalid image url")

        img_pil = Image.open(BytesIO(image_bytes))
        tensor_list.append(app.preprocess(img_pil))

    # classify the batch of images
    img_tensor = torch.stack(tensor_list).to(app.device)
    with torch.no_grad():
        prediction = app.model(img_tensor)
    _, indices = prediction.max(1)

    # extract predicted classes
    results = [
        {"url": image_urls[i], "class": app.labels[class_idx]}
        for i, class_idx in enumerate(indices)
    ]
    json_output = json.dumps(results)

    # save results
    prediction_id = uuid.uuid4()
    s3.put_object(Bucket=app.bucket, Key=f"{app.key}/{prediction_id}.json", Body=json_output)


@app.post("/on-job-complete")
def on_job_complete():
    all_results = []

    # aggregate all classifications
    paginator = s3.get_paginator("list_objects_v2")
    for page in paginator.paginate(Bucket=app.bucket, Prefix=app.key):
        if "Contents" not in page:
            continue
        for obj in page["Contents"]:
            body = s3.get_object(Bucket=app.bucket, Key=obj["Key"])["Body"]
            all_results += json.loads(body.read().decode("utf8"))

    # save single file containing aggregated classifications
    s3.put_object(
        Bucket=app.bucket,
        Key=os.path.join(app.key, "aggregated_results.json"),
        Body=json.dumps(all_results),
    )
