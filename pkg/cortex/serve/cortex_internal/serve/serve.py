# Copyright 2021 Cortex Labs, Inc.
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

import asyncio
import inspect
import json
import os
import signal
import sys
import time
import uuid
import re
import threading
from concurrent.futures import ThreadPoolExecutor
from typing import Any, Dict

from cortex_internal.lib.telemetry import capture_exception, get_default_tags, init_sentry
from cortex_internal.lib.log import configure_logger

init_sentry(tags=get_default_tags())
logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])

from cortex_internal.lib import util
from cortex_internal.lib.api import get_api
from cortex_internal.lib.api.batching import DynamicBatcher
from cortex_internal.lib.concurrency import FileLock, LockedFile
from cortex_internal.lib.exceptions import UserRuntimeException
from fastapi import FastAPI
from fastapi.exceptions import RequestValidationError
from starlette.exceptions import HTTPException as StarletteHTTPException
from starlette.requests import Request
from starlette.responses import JSONResponse, PlainTextResponse, Response

API_SUMMARY_MESSAGE = (
    "make a prediction by sending a post request to this endpoint with a json payload"
)

NANOSECONDS_IN_SECOND = 1e9


request_thread_pool = ThreadPoolExecutor(max_workers=int(os.environ["CORTEX_THREADS_PER_PROCESS"]))
loop = asyncio.get_event_loop()
loop.set_default_executor(request_thread_pool)

app = FastAPI()

local_cache: Dict[str, Any] = {
    "api": None,
    "provider": None,
    "predictor_impl": None,
    "dynamic_batcher": None,
    "predict_route": None,
    "client": None,
}


@app.on_event("startup")
def startup():
    open(f"/mnt/workspace/proc-{os.getpid()}-ready.txt", "a").close()


@app.on_event("shutdown")
def shutdown():
    try:
        os.remove("/mnt/workspace/api_readiness.txt")
    except FileNotFoundError:
        pass

    try:
        os.remove(f"/mnt/workspace/proc-{os.getpid()}-ready.txt")
    except FileNotFoundError:
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
                if "x-request-id" in request.headers:
                    request_id = request.headers["x-request-id"]
                else:
                    request_id = uuid.uuid1()
                file_id = f"/mnt/requests/{request_id}"
                open(file_id, "a").close()

        response = await call_next(request)
    finally:
        if file_id is not None:
            try:
                os.remove(file_id)
            except FileNotFoundError:
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

    if content_type.startswith("text/plain"):
        try:
            charset = "utf-8"
            matches = re.findall(r"charset=(\S+)", content_type)
            if len(matches) > 0:
                charset = matches[-1].rstrip(";")
            body = await request.body()
            request.state.payload = body.decode(charset)
        except Exception as e:
            return PlainTextResponse(content=str(e), status_code=400)
    elif content_type.startswith("multipart/form") or content_type.startswith(
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
    predictor_impl = local_cache["predictor_impl"]
    dynamic_batcher = local_cache["dynamic_batcher"]
    kwargs = build_predict_kwargs(request)

    if dynamic_batcher:
        prediction = dynamic_batcher.predict(**kwargs)
    else:
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
                "please return an object that is JSON serializable (including its nested fields), a bytes object, "
                "a string, or a starlette.response.Response object",
            ) from e
        response = Response(content=json_string, media_type="application/json")

    if util.has_method(predictor_impl, "post_predict"):
        kwargs = build_post_predict_kwargs(prediction, request)
        request_thread_pool.submit(predictor_impl.post_predict, **kwargs)

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

    if hasattr(local_cache["client"], "metadata"):
        client = local_cache["client"]
        predictor = local_cache["api"].predictor
        response["model_metadata"] = client.metadata

    return response


# this exists so that the user's __init__() can be executed by the request thread pool, which helps
# to avoid errors that occur when the user's __init__() function must be called by the same thread
# which executes predict(). This only avoids errors if threads_per_worker == 1
def start():
    future = request_thread_pool.submit(start_fn)
    return future.result()


def start_fn():
    provider = os.environ["CORTEX_PROVIDER"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]
    spec_path = os.environ["CORTEX_API_SPEC"]

    model_dir = os.getenv("CORTEX_MODEL_DIR")
    cache_dir = os.getenv("CORTEX_CACHE_DIR")
    region = os.getenv("AWS_REGION")

    tf_serving_port = os.getenv("CORTEX_TF_BASE_SERVING_PORT", "9000")
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")

    try:
        has_multiple_servers = os.getenv("CORTEX_MULTIPLE_TF_SERVERS")
        if has_multiple_servers:
            with LockedFile("/run/used_ports.json", "r+") as f:
                used_ports = json.load(f)
                for port in used_ports.keys():
                    if not used_ports[port]:
                        tf_serving_port = port
                        used_ports[port] = True
                        break
                f.seek(0)
                json.dump(used_ports, f)
                f.truncate()

        api = get_api(provider, spec_path, model_dir, cache_dir, region)

        client = api.predictor.initialize_client(
            tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
        )

        with FileLock("/run/init_stagger.lock"):
            logger.info("loading the predictor from {}".format(api.predictor.path))
            predictor_impl = api.predictor.initialize_impl(project_dir, client)

        # crons only stop if an unhandled exception occurs
        def check_if_crons_have_failed():
            while True:
                for cron in api.predictor.crons:
                    if not cron.is_alive():
                        os.kill(os.getpid(), signal.SIGQUIT)
                time.sleep(1)

        threading.Thread(target=check_if_crons_have_failed, daemon=True).start()

        local_cache["api"] = api
        local_cache["provider"] = provider
        local_cache["client"] = client
        local_cache["predictor_impl"] = predictor_impl
        local_cache["predict_fn_args"] = inspect.getfullargspec(predictor_impl.predict).args

        if api.python_server_side_batching_enabled:
            dynamic_batching_config = api.api_spec["predictor"]["server_side_batching"]
            local_cache["dynamic_batcher"] = DynamicBatcher(
                predictor_impl,
                max_batch_size=dynamic_batching_config["max_batch_size"],
                batch_interval=dynamic_batching_config["batch_interval"]
                / NANOSECONDS_IN_SECOND,  # convert nanoseconds to seconds
            )

        if util.has_method(predictor_impl, "post_predict"):
            local_cache["post_predict_fn_args"] = inspect.getfullargspec(
                predictor_impl.post_predict
            ).args

        predict_route = "/predict"
        local_cache["predict_route"] = predict_route

    except (UserRuntimeException, Exception) as err:
        if not isinstance(err, UserRuntimeException):
            capture_exception(err)
        logger.exception("failed to start api")
        sys.exit(1)

    app.add_api_route(local_cache["predict_route"], predict, methods=["POST"])
    app.add_api_route(local_cache["predict_route"], get_summary, methods=["GET"])

    return app
