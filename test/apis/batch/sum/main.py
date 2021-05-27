import os
import boto3
import json
import re

from typing import List
from pydantic import BaseModel
from fastapi import FastAPI, Response, status


class Request(BaseModel):
    payload: List[List[int]]


state = {
    "ready": False,
    "bucket": None,
    "key": None,
    "numbers_list": [],
}
app = FastAPI()
s3 = boto3.client("s3")


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
    if len(config.get("dest_s3_dir", "")) == 0:
        raise Exception("'dest_s3_dir' field was not provided in job submission")

    # s3 info
    state["bucket"], state["key"] = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
    state["key"] = os.path.join(state["key"], job_id)

    state["ready"] = True


@app.get("/healthz")
def healthz(response: Response):
    if not state["ready"]:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE


@app.post("/")
def handle_batch(request: Request):
    global state
    for numbers_list in request.payload:
        state["numbers_list"].append(sum(numbers_list))


@app.post("/on-job-complete")
def on_job_complete():
    json_output = json.dumps(state["numbers_list"])
    s3.put_object(Bucket=state["bucket"], Key=f"{state['key']}.json", Body=json_output)
