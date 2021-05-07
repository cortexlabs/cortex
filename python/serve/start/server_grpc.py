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

import importlib
import inspect
import json
import os
import pathlib
import signal
import sys
import threading
import time
import traceback
import uuid
from concurrent import futures
from typing import Callable, Dict, Any, List

import datadog
import grpc
from grpc_reflection.v1alpha import reflection

from cortex_internal.lib.api import RealtimeAPI
from cortex_internal.lib.concurrency import FileLock, LockedFile
from cortex_internal.lib.exceptions import UserRuntimeException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.metrics import MetricsClient
from cortex_internal.lib.telemetry import capture_exception, get_default_tags, init_sentry

NANOSECONDS_IN_SECOND = 1e9


class ThreadPoolExecutorWithRequestMonitor:
    def __init__(self, post_latency_metrics_fn: Callable[[int, float], None], *args, **kwargs):
        self._post_latency_metrics_fn = post_latency_metrics_fn
        self._thread_pool_executor = futures.ThreadPoolExecutor(*args, **kwargs)

    def submit(self, fn, *args, **kwargs):
        request_id = uuid.uuid1()
        file_id = f"/mnt/requests/{request_id}"
        open(file_id, "a").close()

        start_time = time.time()

        def wrapper_fn(*args, **kwargs):
            try:
                result = fn(*args, **kwargs)
            except:
                raise
            finally:
                try:
                    os.remove(file_id)
                except FileNotFoundError:
                    pass
                self._post_latency_metrics_fn(time.time() - start_time)

            return result

        self._thread_pool_executor.submit(wrapper_fn, *args, **kwargs)

    def map(self, *args, **kwargs):
        return self._thread_pool_executor.map(*args, **kwargs)

    def shutdown(self, *args, **kwargs):
        return self._thread_pool_executor.shutdown(*args, **kwargs)


def get_service_name_from_module(module_proto_pb2_grpc) -> Any:
    classes = inspect.getmembers(module_proto_pb2_grpc, inspect.isclass)
    for class_name, _ in classes:
        if class_name.endswith("Servicer"):
            return class_name[: -len("Servicer")]
    # this line will never be reached because we're guaranteed to have one servicer class in the module


def get_servicer_from_module(module_proto_pb2_grpc) -> Any:
    classes = inspect.getmembers(module_proto_pb2_grpc, inspect.isclass)
    for class_name, module_class in classes:
        if class_name.endswith("Servicer"):
            return module_class
    # this line will never be reached because we're guaranteed to have one servicer class in the module


def get_servicer_to_server_from_module(module_proto_pb2_grpc) -> Any:
    functions = inspect.getmembers(module_proto_pb2_grpc, inspect.isfunction)
    for function_name, function in functions:
        if function_name.endswith("_to_server"):
            return function
    # this line will never be reached because we're guaranteed to have one servicer adder in the module


def get_rpc_methods_from_servicer(servicer: Any) -> List[str]:
    rpc_names = []
    for (rpc_name, rpc_method) in inspect.getmembers(servicer, inspect.isfunction):
        rpc_names.append(rpc_name)
    return rpc_names


def build_method_kwargs(method_fn_args, payload, context) -> Dict[str, Any]:
    method_kwargs = {}
    if "payload" in method_fn_args:
        method_kwargs["payload"] = payload
    if "context" in method_fn_args:
        method_kwargs["context"] = context
    return method_kwargs


def construct_handler_servicer_class(ServicerClass: Any, handler_impl: Any) -> Any:
    class HandlerServicer(ServicerClass):
        def __init__(self, handler_impl: Any, api: RealtimeAPI):
            self.handler_impl = handler_impl
            self.api = api

    rpc_names = get_rpc_methods_from_servicer(ServicerClass)
    for rpc_name in rpc_names:
        arg_spec = inspect.getfullargspec(getattr(handler_impl, rpc_name)).args

        def _rpc_method(self: HandlerServicer, payload, context):
            try:
                kwargs = build_method_kwargs(arg_spec, payload, context)
                response = getattr(self.handler_impl, rpc_name)(**kwargs)
                self.api.metrics.post_status_code_request_metrics(200)
            except Exception:
                logger.error(traceback.format_exc())
                self.api.metrics.post_status_code_request_metrics(500)
                context.abort(grpc.StatusCode.INTERNAL, "internal server error")
            return response

        setattr(HandlerServicer, rpc_name, _rpc_method)

    return HandlerServicer


def init():
    project_dir = os.environ["CORTEX_PROJECT_DIR"]
    spec_path = os.environ["CORTEX_API_SPEC"]

    model_dir = os.getenv("CORTEX_MODEL_DIR")
    cache_dir = os.getenv("CORTEX_CACHE_DIR")
    region = os.getenv("AWS_DEFAULT_REGION")

    host_ip = os.environ["HOST_IP"]
    tf_serving_port = os.getenv("CORTEX_TF_BASE_SERVING_PORT", "9000")
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")

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

    config: Dict[str, Any] = {
        "api": None,
        "client": None,
        "handler_impl": None,
        "module_proto_pb2_grpc": None,
    }

    proto_without_ext = pathlib.Path(api.protobuf_path).stem
    module_proto_pb2 = importlib.import_module(proto_without_ext + "_pb2")
    module_proto_pb2_grpc = importlib.import_module(proto_without_ext + "_pb2_grpc")

    client = api.initialize_client(tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port)

    ServicerClass = get_servicer_from_module(module_proto_pb2_grpc)
    rpc_names = get_rpc_methods_from_servicer(ServicerClass)

    with FileLock("/run/init_stagger.lock"):
        logger.info("loading the handler from {}".format(api.path))
        handler_impl = api.initialize_impl(
            project_dir=project_dir,
            client=client,
            metrics_client=MetricsClient(statsd_client),
            proto_module_pb2=module_proto_pb2,
            rpc_method_names=rpc_names,
        )

    # crons only stop if an unhandled exception occurs
    def check_if_crons_have_failed():
        while True:
            for cron in api.crons:
                if not cron.is_alive():
                    os.kill(os.getpid(), signal.SIGQUIT)
            time.sleep(1)

    threading.Thread(target=check_if_crons_have_failed, daemon=True).start()

    HandlerServicer = construct_handler_servicer_class(ServicerClass, handler_impl)

    config["api"] = api
    config["client"] = client
    config["handler_impl"] = handler_impl
    config["module_proto_pb2"] = module_proto_pb2
    config["module_proto_pb2_grpc"] = module_proto_pb2_grpc
    config["handler_servicer"] = HandlerServicer

    return config


def main():
    address = sys.argv[1]
    threads_per_process = int(os.environ["CORTEX_THREADS_PER_PROCESS"])

    try:
        config = init()
    except Exception as err:
        if not isinstance(err, UserRuntimeException):
            capture_exception(err)
        logger.exception("failed to start api")
        sys.exit(1)

    module_proto_pb2 = config["module_proto_pb2"]
    module_proto_pb2_grpc = config["module_proto_pb2_grpc"]
    HandlerServicer = config["handler_servicer"]

    api: RealtimeAPI = config["api"]
    handler_impl = config["handler_impl"]

    server = grpc.server(
        ThreadPoolExecutorWithRequestMonitor(
            post_latency_metrics_fn=api.metrics.post_latency_request_metrics,
            max_workers=threads_per_process,
        ),
        options=[("grpc.max_send_message_length", -1), ("grpc.max_receive_message_length", -1)],
    )

    add_HandlerServicer_to_server = get_servicer_to_server_from_module(module_proto_pb2_grpc)
    add_HandlerServicer_to_server(HandlerServicer(handler_impl, api), server)

    service_name = get_service_name_from_module(module_proto_pb2_grpc)
    SERVICE_NAMES = (
        module_proto_pb2.DESCRIPTOR.services_by_name[service_name].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)

    server.add_insecure_port(address)
    server.start()

    time.sleep(5.0)
    open(f"/mnt/workspace/proc-{os.getpid()}-ready.txt", "a").close()
    server.wait_for_termination()


if __name__ == "__main__":
    init_sentry(tags=get_default_tags())
    logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])
    main()
