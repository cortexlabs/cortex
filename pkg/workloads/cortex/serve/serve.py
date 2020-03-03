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
import json
from concurrent.futures import ThreadPoolExecutor
import threading
import math
import asyncio

from fastapi import FastAPI
from fastapi.exceptions import RequestValidationError
from starlette.requests import Request
from starlette.responses import Response
from starlette.background import BackgroundTasks
from starlette.exceptions import HTTPException as StarletteHTTPException

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


def is_prediction_request(request):
    return request.url.path == "/predict" and request.method == "POST"


@app.exception_handler(StarletteHTTPException)
async def http_exception_handler(request, e):
    response = Response(content=str(e.detail), status_code=e.status_code)
    apply_cors_headers(request, response)
    return response


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request, e):
    response = Response(content=str(e), status_code=400)
    apply_cors_headers(request, response)
    return response


@app.exception_handler(Exception)
async def uncaught_exception_handler(request, e):
    response = Response(content="internal server error", status_code=500)
    apply_cors_headers(request, response)
    return response


@app.middleware("http")
async def register_request(request: Request, call_next):
    request.state.start_time = time.time()

    file_id = None
    response = None
    try:
        if is_prediction_request(request):
            request_id = request.headers["x-request-id"]
            file_id = f"/mnt/requests/{request_id}"
            open(file_id, "a").close()

        response = await call_next(request)
    finally:
        if file_id is not None:
            try:
                os.remove(file_id)
            except:
                pass

        if is_prediction_request(request):
            status_code = 500
            if response is not None:
                status_code = response.status_code
                apply_cors_headers(request, response)
            api = local_cache["api"]
            api.post_request_metrics(status_code, time.time() - request.state.start_time)

    return response


def apply_cors_headers(request: Request, response: Response):
    response.headers["Access-Control-Allow-Origin"] = "*"
    response.headers["Access-Control-Allow-Headers"] = request.headers.get(
        "Access-Control-Request-Headers", "*"
    )


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

    response = Response(content=json_string, media_type="application/json")

    if api.tracker is not None:
        try:
            predicted_value = api.tracker.extract_predicted_value(prediction)
            api.post_tracker_metrics(predicted_value)
            if (
                api.tracker.model_type == "classification"
                and predicted_value not in local_cache["class_set"]
            ):
                tasks = BackgroundTasks()
                tasks.add_task(api.upload_class, class_name=predicted_value)
                local_cache["class_set"].add(predicted_value)
                response.background = tasks
        except:
            cx_logger().warn("unable to record prediction metric", exc_info=True)

    return response


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
