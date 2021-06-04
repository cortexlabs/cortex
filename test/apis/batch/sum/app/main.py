import os
import boto3
import json
import re

from typing import List
from fastapi import FastAPI, status
from fastapi.responses import PlainTextResponse

app = FastAPI()
app.ready = False
app.numbers_list = []
s3 = boto3.client("s3")


@app.on_event("startup")
def startup():
    # read job spec
    with open("/cortex/spec/job.json", "r") as f:
        job_spec = json.load(f)
    print(json.dumps(job_spec, indent=2))

    # get metadata
    config = job_spec["config"]
    job_id = job_spec["job_id"]
    if len(config.get("dest_s3_dir", "")) == 0:
        raise Exception("'dest_s3_dir' field was not provided in job submission")

    # s3 info
    app.bucket, app.key = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
    app.key = os.path.join(app.key, job_id)

    app.ready = True


@app.get("/healthz")
def healthz():
    if app.ready:
        return PlainTextResponse("ok")
    return PlainTextResponse("service unavailable", status_code=status.HTTP_503_SERVICE_UNAVAILABLE)


@app.post("/")
def handle_batch(batches: List[List[int]]):
    for numbers_list in batches:
        app.numbers_list.append(sum(numbers_list))


@app.post("/on-job-complete")
def on_job_complete():
    # this is only intended to work if 1 worker is used (since on-job-complete runs once across all workers)
    json_output = json.dumps(app.numbers_list)
    s3.put_object(Bucket=app.bucket, Key=f"{app.key}.json", Body=json_output)
