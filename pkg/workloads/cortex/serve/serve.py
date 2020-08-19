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
import inspect
import time
import json
from concurrent.futures import ThreadPoolExecutor
import threading
import math
import asyncio
from typing import Any

from fastapi import Body, FastAPI
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from starlette.requests import Request
from starlette.responses import Response, PlainTextResponse, JSONResponse
from starlette.background import BackgroundTasks
from starlette.exceptions import HTTPException as StarletteHTTPException

from cortex.lib import util
from cortex.lib.type import API, get_spec
from cortex.lib.log import cx_logger
from cortex.lib.storage import S3, LocalStorage, FileLock
from cortex.lib.exceptions import UserRuntimeException

API_SUMMARY_MESSAGE = (
    "make a prediction by sending a post request to this endpoint with a json payload"
)

API_LIVENESS_UPDATE_PERIOD = 5  # seconds


request_thread_pool = ThreadPoolExecutor(max_workers=int(os.environ["CORTEX_THREADS_PER_PROCESS"]))
loop = asyncio.get_event_loop()
loop.set_default_executor(request_thread_pool)

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

local_cache = {
    "api": None,
    "provider": None,
    "predictor_impl": None,
    "predict_route": None,
    "client": None,
    "class_set": set(),
}


def update_api_liveness():
    threading.Timer(API_LIVENESS_UPDATE_PERIOD, update_api_liveness).start()
    with open("/mnt/workspace/api_liveness.txt", "w") as f:
        f.write(str(math.ceil(time.time())))


@app.on_event("startup")
def startup():
    open("/mnt/workspace/api_readiness.txt", "a").close()
    update_api_liveness()


@app.on_event("shutdown")
def shutdown():
    try:
        os.remove("/mnt/workspace/api_readiness.txt")
    except:
        pass

    try:
        os.remove("/mnt/workspace/api_liveness.txt")
    except:
        pass


def is_prediction_request(request):
    return request.url.path == local_cache["predict_route"] and request.method == "POST"


@app.exception_handler(StarletteHTTPException)
async def http_exception_handler(request, e):
    response = Response(content=str(e.detail), status_code=e.status_code)
    return response


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request, e):
    response = Response(content=str(e), status_code=400)
    return response


@app.exception_handler(Exception)
async def uncaught_exception_handler(request, e):
    response = Response(content="internal server error", status_code=500)
    return response


@app.middleware("http")
async def register_request(request: Request, call_next):
    request.state.start_time = time.time()

    file_id = None
    response = None
    try:
        if is_prediction_request(request):
            if local_cache["provider"] != "local":
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
            api = local_cache["api"]
            api.post_request_metrics(status_code, time.time() - request.state.start_time)

    return response


@app.middleware("http")
async def parse_payload(request: Request, call_next):
    if not is_prediction_request(request):
        return await call_next(request)

    if "payload" not in local_cache["predict_fn_args"]:
        return await call_next(request)

    content_type = request.headers.get("content-type", "").lower()

    if content_type.startswith("multipart/form") or content_type.startswith(
        "application/x-www-form-urlencoded"
    ):
        try:
            request.state.payload = await request.form()
        except Exception as e:
            return PlainTextResponse(content=str(e), status_code=400)
    elif content_type.startswith("application/json"):
        try:
            request.state.payload = await request.json()
        except json.JSONDecodeError as e:
            return JSONResponse(content={"error": str(e)}, status_code=400)
    else:
        request.state.payload = await request.body()

    return await call_next(request)


def predict(request: Request):
    tasks = BackgroundTasks()
    api = local_cache["api"]
    predictor_impl = local_cache["predictor_impl"]
    kwargs = build_predict_kwargs(request)

    prediction = predictor_impl.predict(**kwargs)

    if isinstance(prediction, bytes):
        response = Response(content=prediction, media_type="application/octet-stream")
    elif isinstance(prediction, str):
        response = Response(content=prediction, media_type="text/plain")
    elif isinstance(prediction, Response):
        response = prediction
    else:
        try:
            json_string = json.dumps(prediction)
        except Exception as e:
            raise UserRuntimeException(
                str(e),
                "please return an object that is JSON serializable (including its nested fields), a bytes object, a string, or a starlette.response.Response object",
            ) from e
        response = Response(content=json_string, media_type="application/json")

    if local_cache["provider"] != "local" and api.monitoring is not None:
        try:
            predicted_value = api.monitoring.extract_predicted_value(prediction)
            api.post_monitoring_metrics(predicted_value)
            if (
                api.monitoring.model_type == "classification"
                and predicted_value not in local_cache["class_set"]
            ):
                tasks.add_task(api.upload_class, class_name=predicted_value)
                local_cache["class_set"].add(predicted_value)
        except:
            cx_logger().warn("unable to record prediction metric", exc_info=True)

    if util.has_method(predictor_impl, "post_predict"):
        kwargs = build_post_predict_kwargs(prediction, request)
        tasks.add_task(predictor_impl.post_predict, **kwargs)

    if len(tasks.tasks) > 0:
        response.background = tasks

    return response


def build_predict_kwargs(request: Request):
    kwargs = {}

    if "payload" in local_cache["predict_fn_args"]:
        kwargs["payload"] = request.state.payload
    if "headers" in local_cache["predict_fn_args"]:
        kwargs["headers"] = request.headers
    if "query_params" in local_cache["predict_fn_args"]:
        kwargs["query_params"] = request.query_params
    if "batch_id" in local_cache["predict_fn_args"]:
        kwargs["batch_id"] = None

    return kwargs


def build_post_predict_kwargs(response, request: Request):
    kwargs = {}

    if "payload" in local_cache["post_predict_fn_args"]:
        kwargs["payload"] = request.state.payload
    if "headers" in local_cache["post_predict_fn_args"]:
        kwargs["headers"] = request.headers
    if "query_params" in local_cache["post_predict_fn_args"]:
        kwargs["query_params"] = request.query_params
    if "response" in local_cache["post_predict_fn_args"]:
        kwargs["response"] = response

    return kwargs


def get_summary():
    response = {"message": API_SUMMARY_MESSAGE}

    if hasattr(local_cache["client"], "input_signatures"):
        response["model_signatures"] = local_cache["client"].input_signatures

    return response


# this exists so that the user's __init__() can be executed by the request thread pool, which helps
# to avoid errors that occur when the user's __init__() function must be called by the same thread
# which executes predict(). This only avoids errors if threads_per_worker == 1
def start():
    future = request_thread_pool.submit(start_fn)
    return future.result()


def start_fn():
    cache_dir = os.environ["CORTEX_CACHE_DIR"]
    provider = os.environ["CORTEX_PROVIDER"]
    spec_path = os.environ["CORTEX_API_SPEC"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]

    model_dir = os.getenv("CORTEX_MODEL_DIR")
    tf_serving_port = os.getenv("CORTEX_TF_BASE_SERVING_PORT", "9000")
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")

    if provider == "local":
        storage = LocalStorage(os.getenv("CORTEX_CACHE_DIR"))
    else:
        storage = S3(bucket=os.environ["CORTEX_BUCKET"], region=os.environ["AWS_REGION"])

    has_multiple_servers = os.getenv("CORTEX_MULTIPLE_TF_SERVERS")
    if has_multiple_servers:
        with FileLock("/run/used_ports.json.lock"):
            with open("/run/used_ports.json", "r+") as f:
                used_ports = json.load(f)
                for port in used_ports.keys():
                    if not used_ports[port]:
                        tf_serving_port = port
                        used_ports[port] = True
                        break
                f.seek(0)
                json.dump(used_ports, f)
                f.truncate()

    try:
        raw_api_spec = get_spec(provider, storage, cache_dir, spec_path)
        api = API(
            provider=provider,
            storage=storage,
            model_dir=model_dir,
            cache_dir=cache_dir,
            **raw_api_spec,
        )
        client = api.predictor.initialize_client(
            tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
        )
        cx_logger().info("loading the predictor from {}".format(api.predictor.path))
        predictor_impl = api.predictor.initialize_impl(project_dir, client, raw_api_spec, None)

        local_cache["api"] = api
        local_cache["provider"] = provider
        local_cache["client"] = client
        local_cache["predictor_impl"] = predictor_impl
        local_cache["predict_fn_args"] = inspect.getfullargspec(predictor_impl.predict).args
        if util.has_method(predictor_impl, "post_predict"):
            local_cache["post_predict_fn_args"] = inspect.getfullargspec(
                predictor_impl.post_predict
            ).args

        predict_route = "/"
        if provider != "local":
            predict_route = "/predict"
        local_cache["predict_route"] = predict_route
    except:
        cx_logger().exception("failed to start api")
        sys.exit(1)

    if (
        provider != "local"
        and api.monitoring is not None
        and api.monitoring.model_type == "classification"
    ):
        try:
            local_cache["class_set"] = api.get_cached_classes()
        except:
            cx_logger().warn("an error occurred while attempting to load classes", exc_info=True)

    app.add_api_route(local_cache["predict_route"], predict, methods=["POST"])
    app.add_api_route(local_cache["predict_route"], get_summary, methods=["GET"])

    return app
