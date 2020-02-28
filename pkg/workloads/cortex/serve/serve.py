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
import asyncio

from fastapi import FastAPI
from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.requests import Request
from starlette.responses import Response

from cortex import consts
from cortex.lib import util
from cortex.lib.type import API
from cortex.lib.log import cx_logger, debug_obj
from cortex.lib.storage import S3
from cortex.lib.exceptions import UserRuntimeException


loop = asyncio.get_event_loop()
loop.set_default_executor(ThreadPoolExecutor(max_workers=int(os.environ["THREADS"])))

logger = logging.getLogger("api")

app = FastAPI()


API_SUMMARY_MESSAGE = (
    "make a prediction by sending a post request to this endpoint with a json payload"
)

local_cache = {"api": None, "predictor_impl": None, "client": None, "class_set": set()}


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
        except Exception as e:
            cx_logger().warn("an error occurred while attempting to load classes", exc_info=True)

    open("/mnt/health_check.txt", "a").close()
    return app


@app.middleware("http")
async def my_middleware(request: Request, call_next):
    pid = os.getpid()
    rid = uuid.uuid4()
    file_id = f"/mnt/requests/{pid}.{rid}"
    open(file_id, "a").close()
    request_start_time = time.time()

    response = await call_next(request)

    os.remove(file_id)
    after_request(request, response, request_start_time)
    return response


@app.post("/predict")
def predict(request: dict, debug=False):
    api = local_cache["api"]
    predictor_impl = local_cache["predictor_impl"]

    try:
        debug_obj("payload", request, debug)
        try:
            output = predictor_impl.predict(request)
        except Exception as e:
            raise UserRuntimeException(api.predictor.path, "predict", str(e)) from e
        debug_obj("prediction", output, debug)
    except Exception as e:
        cx_logger().exception("prediction failed")
        return prediction_failed(str(e))

    try:
        if api.tracker is not None:
            predicted_value = api.tracker.extract_predicted_value(output)
            api.post_tracker_metrics(predicted_value)
            if predicted_value is not None and predicted_value not in local_cache["class_set"]:
                api.upload_class(predicted_value)
                local_cache["class_set"].add(predicted_value)
    except Exception as e:
        cx_logger().warn("unable to record prediction metric", exc_info=True)

    return output


@app.get("/predict")
def get_summary():
    response = {"message": API_SUMMARY_MESSAGE}

    if hasattr(local_cache["client"], "input_signature"):
        response["model_signature"] = local_cache["client"].input_signature
    return response


def prediction_failed(reason):
    message = "prediction failed: {}".format(reason)
    cx_logger().error(message)
    return message


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


def after_request(request: Request, response: Response, start_time: float):
    response.headers["Access-Control-Allow-Origin"] = "*"
    response.headers["Access-Control-Allow-Headers"] = request.headers.get(
        "Access-Control-Request-Headers", "*"
    )

    if not (request.url.path == "/predict" and request.method == "POST"):
        return response

    api = local_cache["api"]
    api.post_latency_metrics(response.status_code, start_time)

    return response
