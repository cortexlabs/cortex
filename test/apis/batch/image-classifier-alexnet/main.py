import os, json, re
from typing import Any, List

from fastapi import FastAPI, Response, status
from pydantic import BaseModel

import requests
import torch
import torchvision
from torchvision import transforms
from PIL import Image
from io import BytesIO
import boto3


class Request(BaseModel):
    payload: List[Any]


state = {
    "ready": False,
    "model": None,
    "preprocess": None,
    "job_id": None,
    "labels": None,
    "bucket": None,
    "key": None,
}
device = os.getenv("TARGET_DEVICE", "cpu")
s3 = boto3.client("s3")
app = FastAPI()


@app.on_event("startup")
def startup():
    global state

    # read job spec
    with open("/cortex/spec/job.json", "r") as f:
        job_spec = json.load(f)
    print(json.dumps(job_spec, indent=2))

    # get metadata
    config = job_spec["config"]
    job_id = job_spec["job_id"]
    state["job_id"] = job_spec["job_id"]
    if len(config.get("dest_s3_dir", "")) == 0:
        raise Exception("'dest_s3_dir' field was not provided in job submission")

    # s3 info
    state["bucket"], state["key"] = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
    state["key"] = os.path.join(state["key"], job_id)

    # loading model
    normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
    state["preprocess"] = transforms.Compose(
        [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
    )
    state["labels"] = requests.get(
        "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
    ).text.split("\n")[1:]
    state["model"] = torchvision.models.alexnet(pretrained=True).eval().to(device)

    state["ready"] = True


@app.get("/healthz")
def healthz(response: Response):
    if not state["ready"]:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE


@app.post("/")
def handle_batch(request: Request):
    payload = request.payload
    job_id = state["job_id"]
    tensor_list = []

    # download and preprocess each image
    for image_url in payload:
        if image_url.startswith("s3://"):
            bucket, image_key = re.match("s3://(.+?)/(.+)", image_url).groups()
            image_bytes = s3.get_object(Bucket=bucket, Key=image_key)["Body"].read()
        elif image_url.startswith("http"):
            image_bytes = requests.get(image_url).content
        else:
            raise RuntimeError(f"{image_url}: invalid image url")

        img_pil = Image.open(BytesIO(image_bytes))
        tensor_list.append(state["preprocess"](img_pil))

    # classify the batch of images
    img_tensor = torch.stack(tensor_list)
    with torch.no_grad():
        prediction = state["model"](img_tensor)
    _, indices = prediction.max(1)

    # extract predicted classes
    results = [
        {"url": payload[i], "class": state["labels"][class_idx]}
        for i, class_idx in enumerate(indices)
    ]
    json_output = json.dumps(results)

    # save results
    s3.put_object(
        Bucket=state["bucket"], Key=f"{state['key']}/{job_id}.json", Body=json_output
    )


@app.post("/on-job-complete")
def on_job_complete():
    all_results = []

    # aggregate all classifications
    paginator = s3.get_paginator("list_objects_v2")
    for page in paginator.paginate(Bucket=state["bucket"], Prefix=state["key"]):
        for obj in page["Contents"]:
            body = s3.get_object(Bucket=state["bucket"], Key=obj["Key"])["Body"]
            all_results += json.loads(body.read().decode("utf8"))

    # save single file containing aggregated classifications
    s3.put_object(
        Bucket=state["bucket"],
        Key=os.path.join(state["key"], "aggregated_results.json"),
        Body=json.dumps(all_results),
    )
