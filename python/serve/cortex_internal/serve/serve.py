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
import re
import signal
import sys
import threading
import time
import uuid
from concurrent.futures import ThreadPoolExecutor
from typing import Any, Dict

import datadog
from asgiref.sync import async_to_sync
from fastapi import FastAPI
from fastapi.exceptions import RequestValidationError
from starlette.exceptions import HTTPException as StarletteHTTPException
from starlette.requests import Request
from starlette.responses import JSONResponse, PlainTextResponse, Response

from cortex_internal.lib import util
from cortex_internal.lib.api import DynamicBatcher, RealtimeAPI
from cortex_internal.lib.concurrency import FileLock, LockedFile
from cortex_internal.lib.exceptions import UserException, UserRuntimeException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.metrics import MetricsClient
from cortex_internal.lib.telemetry import capture_exception, get_default_tags, init_sentry

init_sentry(tags=get_default_tags())
logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])

NANOSECONDS_IN_SECOND = 1e9


request_thread_pool = ThreadPoolExecutor(max_workers=int(os.environ["CORTEX_THREADS_PER_PROCESS"]))
loop = asyncio.get_event_loop()
loop.set_default_executor(request_thread_pool)

app = FastAPI()

local_cache: Dict[str, Any] = {
    "api": None,
    "handler_impl": None,
    "dynamic_batcher": None,
    "api_route": None,
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


def is_allowed_request(request):
    return (
        request.url.path == local_cache["api_route"]
        and request.method.lower() in local_cache["handle_fn_args"]
    )


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
        if is_allowed_request(request):
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

        if is_allowed_request(request):
            status_code = 500
            if response is not None:
                status_code = response.status_code
            api: RealtimeAPI = local_cache["api"]
            api.metrics.post_request_metrics(status_code, time.time() - request.state.start_time)

    return response


@app.middleware("http")
async def parse_payload(request: Request, call_next):
    if not is_allowed_request(request):
        return await call_next(request)

    verb = request.method.lower()
    if (
        verb in local_cache["handle_fn_args"]
        and "payload" not in local_cache["handle_fn_args"][verb]
    ):
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


def handle(request: Request):
    if async_to_sync(request.is_disconnected)():
        return Response(status_code=499, content="disconnected client")

    verb = request.method.lower()
    handle_fn_args = local_cache["handle_fn_args"]
    if verb not in handle_fn_args:
        return Response(status_code=405, content="method not implemented")

    handler_impl = local_cache["handler_impl"]
    dynamic_batcher = None
    if verb == "post":
        dynamic_batcher: DynamicBatcher = local_cache["dynamic_batcher"]
    kwargs = build_handler_kwargs(request)

    if dynamic_batcher:
        result = dynamic_batcher.process(**kwargs)
    else:
        result = getattr(handler_impl, f"handle_{verb}")(**kwargs)

    callback = None
    if isinstance(result, tuple) and len(result) == 2 and callable(result[1]):
        callback = result[1]
        result = result[0]

    if isinstance(result, bytes):
        response = Response(content=result, media_type="application/octet-stream")
    elif isinstance(result, str):
        response = Response(content=result, media_type="text/plain")
    elif isinstance(result, Response):
        response = result
    else:
        try:
            json_string = json.dumps(result)
        except Exception as e:
            raise UserRuntimeException(
                str(e),
                "please return an object that is JSON serializable (including its nested fields), a bytes object, "
                "a string, or a `starlette.response.Response` object",
            ) from e
        response = Response(content=json_string, media_type="application/json")

    if callback is not None:
        request_thread_pool.submit(callback)

    return response


def build_handler_kwargs(request: Request):
    kwargs = {}
    verb = request.method.lower()

    if "payload" in local_cache["handle_fn_args"][verb]:
        kwargs["payload"] = request.state.payload
    if "headers" in local_cache["handle_fn_args"][verb]:
        kwargs["headers"] = request.headers
    if "query_params" in local_cache["handle_fn_args"][verb]:
        kwargs["query_params"] = request.query_params

    return kwargs


def get_summary():
    response = {}

    if hasattr(local_cache["client"], "metadata"):
        client = local_cache["client"]
        response = {
            "model_metadata": client.metadata,
        }

    return response


# this exists so that the user's __init__() can be executed by the request thread pool, which helps
# to avoid errors that occur when the user's __init__() function must be called by the same thread
# which executes handle_<verb>() methods. This only avoids errors if threads_per_worker == 1
def start():
    future = request_thread_pool.submit(start_fn)
    return future.result()


def start_fn():
    project_dir = os.environ["CORTEX_PROJECT_DIR"]
    spec_path = os.environ["CORTEX_API_SPEC"]
    model_dir = os.getenv("CORTEX_MODEL_DIR")
    host_ip = os.environ["HOST_IP"]
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

        datadog.initialize(statsd_host=host_ip, statsd_port=9125)
        statsd_client = datadog.statsd

        with open(spec_path) as json_file:
            api_spec = json.load(json_file)
        api = RealtimeAPI(api_spec, statsd_client, model_dir)

        client = api.initialize_client(
            tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
        )

        with FileLock("/run/init_stagger.lock"):
            logger.info("loading the handler from {}".format(api.path))
            handler_impl = api.initialize_impl(
                project_dir=project_dir, client=client, metrics_client=MetricsClient(statsd_client)
            )

        # crons only stop if an unhandled exception occurs
        def check_if_crons_have_failed():
            while True:
                for cron in api.crons:
                    if not cron.is_alive():
                        os.kill(os.getpid(), signal.SIGQUIT)
                time.sleep(1)

        threading.Thread(target=check_if_crons_have_failed, daemon=True).start()

        local_cache["api"] = api
        local_cache["client"] = client
        local_cache["handler_impl"] = handler_impl

        local_cache["handle_fn_args"] = {}
        for verb in ["post", "get", "put", "patch", "delete"]:
            if util.has_method(handler_impl, f"handle_{verb}"):
                local_cache["handle_fn_args"][verb] = inspect.getfullargspec(
                    getattr(handler_impl, f"handle_{verb}")
                ).args
        if len(local_cache["handle_fn_args"]) == 0:
            raise UserException(
                "no user-defined `handle_<verb>` method found in handler class; define at least one verb handler (`handle_post`, `handle_get`, `handle_put`, `handle_patch`, `handle_delete`)"
            )

        if api.python_server_side_batching_enabled:
            dynamic_batching_config = api.api_spec["handler"]["server_side_batching"]

            if "post" in local_cache["handle_fn_args"]:
                local_cache["dynamic_batcher"] = DynamicBatcher(
                    handler_impl,
                    method_name=f"handle_post",
                    max_batch_size=dynamic_batching_config["max_batch_size"],
                    batch_interval=dynamic_batching_config["batch_interval"]
                    / NANOSECONDS_IN_SECOND,  # convert nanoseconds to seconds
                )
            else:
                raise UserException(
                    "dynamic batcher has been enabled, but no `handle_post` method could be found in the `Handler` class"
                )

        local_cache["api_route"] = "/"
        local_cache["info_route"] = "/info"

    except Exception as err:
        if not isinstance(err, UserRuntimeException):
            capture_exception(err)
        logger.exception("failed to start api")
        sys.exit(1)

    app.add_api_route(
        local_cache["api_route"],
        handle,
        methods=[verb.upper() for verb in local_cache["handle_fn_args"]],
    )
    app.add_api_route(local_cache["info_route"], get_summary, methods=["GET"])

    return app
