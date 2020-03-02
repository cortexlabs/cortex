# Copyright 2020 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import sys
import os
import argparse
import time
import logging
import json
import uuid
from concurrent.futures import ThreadPoolExecutor
import threading
import math
import asyncio

from fastapi import FastAPI
from starlette.requests import Request
from starlette.responses import Response, JSONResponse
from starlette.background import BackgroundTasks

from cortex import consts
from cortex.lib import util
from cortex.lib.type import API
from cortex.lib.log import cx_logger, debug_obj
from cortex.lib.storage import S3


API_SUMMARY_MESSAGE = (
    "make a prediction by sending a post request to this endpoint with a json payload"
)

API_LIVENESS_UPDATE_PERIOD = 5  # seconds


loop = asyncio.get_event_loop()
loop.set_default_executor(
    ThreadPoolExecutor(max_workers=int(os.environ["CORTEX_THREADS_PER_WORKER"]))
)

app = FastAPI()
local_cache = {"api": None, "predictor_impl": None, "client": None, "class_set": set()}


def update_api_liveness():
    threading.Timer(API_LIVENESS_UPDATE_PERIOD, update_api_liveness).start()
    with open("/mnt/api_liveness.txt", "w") as f:
        f.write(str(math.ceil(time.time())))


def is_prediction_request(request):
    return request.url.path == "/predict" and request.method == "POST"


@app.on_event("startup")
def startup():
    open("/mnt/api_readiness.txt", "a").close()
    update_api_liveness()


@app.on_event("shutdown")
def shutdown():
    try:
        os.remove("/mnt/api_readiness.txt")
    except:
        pass

    try:
        os.remove("/mnt/api_liveness.txt")
    except:
        pass


@app.middleware("http")
async def register_request(request: Request, call_next):
    request.state.started_time = time.time()

    file_id = None
    try:
        if is_prediction_request(request):
            request_id = request.headers["x-request-id"]
            file_id = f"/mnt/requests/{request_id}"
            start_time = time.time()
            open(file_id, "a").close()

        response = await call_next(request)
        if is_prediction_request(request):
            add_metrics_background_task(request, response)
            apply_cors_headers(request, response)
    except:
        raise
    finally:
        if file_id is not None:
            try:
                os.remove(file_id)
            except:
                pass

    return response


def add_metrics_background_task(request: Request, response: Response):
    if response.background is None:
        response.background = BackgroundTasks()

    response.background.add_task(
        post_response,
        request=request,
        response=response,
        total_time=time.time() - request.state.started_time,
    )

    return response


def post_response(request: Request, response: Response, total_time: float):
    api = local_cache["api"]
    api.post_latency_metrics(response.status_code, total_time)


def apply_cors_headers(request: Request, response: Response):
    response.headers["Access-Control-Allow-Origin"] = "*"
    response.headers["Access-Control-Allow-Headers"] = request.headers.get(
        "Access-Control-Request-Headers", "*"
    )

    return response


@app.post("/predict")
def predict(request: dict, debug=False):
    api = local_cache["api"]
    predictor_impl = local_cache["predictor_impl"]

    debug_obj("payload", request, debug)
    prediction = predictor_impl.predict(request)
    debug_obj("prediction", prediction, debug)

    try:
        json_string = json.dumps(prediction)
    except:
        json_string = util.json_tricks_encoder().encode(prediction)

    tasks = BackgroundTasks()
    if api.tracker is not None:
        tasks.add_task(track_prediction, api=api, prediction=prediction)
    response = Response(content=json_string, media_type="application/json", background=tasks)

    return response


def track_prediction(api, prediction):
    try:
        predicted_value = api.tracker.extract_predicted_value(prediction)
        api.post_tracker_metrics(predicted_value)
        if predicted_value is not None and predicted_value not in local_cache["class_set"]:
            api.upload_class(predicted_value)
            local_cache["class_set"].add(predicted_value)
    except:
        cx_logger().warn("unable to record prediction metric", exc_info=True)


@app.get("/predict")
def get_summary():
    response = {"message": API_SUMMARY_MESSAGE}

    if hasattr(local_cache["client"], "input_signature"):
        response["model_signature"] = local_cache["client"].input_signature
    return response


def assert_api_version():
    if os.environ["CORTEX_VERSION"] != consts.CORTEX_VERSION:
        raise ValueError(
            "api version mismatch (operator: {}, image: {})".format(
                os.environ["CORTEX_VERSION"], consts.CORTEX_VERSION
            )
        )


def get_spec(storage, cache_dir, s3_path):
    local_spec_path = os.path.join(cache_dir, "api_spec.msgpack")
    _, key = S3.deconstruct_s3_path(s3_path)
    storage.download_file(key, local_spec_path)
    return util.read_msgpack(local_spec_path)


def start():
    cache_dir = os.environ["CORTEX_CACHE_DIR"]
    spec = os.environ["CORTEX_API_SPEC"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]
    model_dir = os.getenv("CORTEX_MODEL_DIR", None)
    tf_serving_port = os.getenv("CORTEX_TF_SERVING_PORT", None)
    storage = S3(bucket=os.environ["CORTEX_BUCKET"], region=os.environ["AWS_REGION"])

    try:
        raw_api_spec = get_spec(storage, cache_dir, spec)
        api = API(storage=storage, cache_dir=cache_dir, **raw_api_spec)
        client = api.predictor.initialize_client(model_dir, tf_serving_port)
        cx_logger().info("loading the predictor from {}".format(api.predictor.path))
        predictor_impl = api.predictor.initialize_impl(project_dir, client)

        local_cache["api"] = api
        local_cache["client"] = client
        local_cache["predictor_impl"] = predictor_impl
    except:
        cx_logger().exception("failed to start api")
        sys.exit(1)

    if api.tracker is not None and api.tracker.model_type == "classification":
        try:
            local_cache["class_set"] = api.get_cached_classes()
        except:
            cx_logger().warn("an error occurred while attempting to load classes", exc_info=True)

    return app
