import os
import sys
import json
import time
import uuid
import signal
import threading
import traceback
import pathlib
import importlib
import inspect
from typing import Callable, Dict, Any
from concurrent import futures

import grpc

from cortex_internal.lib.api import get_api
from cortex_internal.lib.concurrency import FileLock, LockedFile
from cortex_internal.lib.exceptions import UserRuntimeException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.metrics import MetricsClient
from cortex_internal.lib.telemetry import capture_exception, get_default_tags, init_sentry

NANOSECONDS_IN_SECOND = 1e9


class ThreadPoolExecutorWithRequestMonitor:
    def __init__(self, post_metrics_fn: Callable[[int, float]], *args, **kwargs):
        self._post_metrics_fn = post_metrics_fn
        self._thread_pool_executor = futures.ThreadPoolExecutor(*args, **kwargs)

    def submit(self, fn, *args, **kwargs):
        request_id = uuid.uuid1()
        file_id = f"/mnt/requests/{request_id}"
        open(file_id, "a").close()

        start_time = time.time()

        def wrapper_fn(*args, **kwargs):
            successful_execution = False
            try:
                result = fn(*args, **kwargs)
                successful_execution = True
            except:
                raise
            finally:
                try:
                    os.remove(file_id)
                except FileNotFoundError:
                    pass
                if successful_execution:
                    status_code = 200
                else:
                    status_code = 500
                self._post_metrics_fn(status_code, time.time() - start_time)

            return result

        self._thread_pool_executor.submit(wrapper_fn, *args, **kwargs)

    def map(self, *args, **kwargs):
        return self._thread_pool_executor.map(*args, **kwargs)

    def shutdown(self, *args, **kwargs):
        return self._thread_pool_executor.shutdown(*args, **kwargs)


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


def build_predict_kwargs(predict_fn_args, payload, context) -> Dict[str, Any]:
    predict_kwargs = {}
    if "payload" in predict_fn_args:
        predict_kwargs["payload"] = payload
    if "context" in predict_fn_args:
        predict_kwargs["context"] = context
    return predict_kwargs


def init():
    provider = os.environ["CORTEX_PROVIDER"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]
    spec_path = os.environ["CORTEX_API_SPEC"]

    model_dir = os.getenv("CORTEX_MODEL_DIR")
    cache_dir = os.getenv("CORTEX_CACHE_DIR")
    region = os.getenv("AWS_REGION")

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

    api = get_api(provider, spec_path, model_dir, cache_dir, region)

    config: Dict[str, Any] = {
        "api": None,
        "provider": None,
        "client": None,
        "predictor_impl": None,
        "module_proto_pb2_grpc": None,
    }

    proto_without_ext = pathlib.Path(api.predictor.protobuf_path).stem
    module_proto_pb2 = importlib.import_module(proto_without_ext + "_pb2")
    module_proto_pb2_grpc = importlib.import_module(proto_without_ext + "_pb2_grpc")

    client = api.predictor.initialize_client(
        tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
    )

    with FileLock("/run/init_stagger.lock"):
        logger.info("loading the predictor from {}".format(api.predictor.path))
        metrics_client = MetricsClient(api.statsd)
        predictor_impl = api.predictor.initialize_impl(
            project_dir=project_dir,
            client=client,
            metrics_client=metrics_client,
            proto_module_pb2=module_proto_pb2,
        )

    # crons only stop if an unhandled exception occurs
    def check_if_crons_have_failed():
        while True:
            for cron in api.predictor.crons:
                if not cron.is_alive():
                    os.kill(os.getpid(), signal.SIGQUIT)
            time.sleep(1)

    threading.Thread(target=check_if_crons_have_failed, daemon=True).start()

    ServicerClass = get_servicer_from_module(module_proto_pb2_grpc)

    class PredictorServicer(ServicerClass):
        def __init__(self, config: Dict[str, Any]):
            self.config = config

        def Predict(self, payload, context):
            try:
                kwargs = build_predict_kwargs(self.config["predict_fn_args"], payload, context)
                response = self.config["predictor_impl"].predict(**kwargs)
            except Exception:
                logger.error(traceback.format_exc())
                context.abort(grpc.StatusCode.INTERNAL, "internal server error")
            return response

    config["api"] = api
    config["provider"] = provider
    config["client"] = client
    config["predictor_impl"] = predictor_impl
    config["predict_fn_args"] = inspect.getfullargspec(predictor_impl.predict).args
    config["module_proto_pb2_grpc"] = module_proto_pb2_grpc
    config["predictor_servicer"] = PredictorServicer

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

    module_proto_pb2_grpc = config["module_proto_pb2_grpc"]
    PredictorServicer = config["predictor_servicer"]
    api = config["api"]

    server = grpc.server(
        ThreadPoolExecutorWithRequestMonitor(
            post_metrics_fn=api.post_request_metrics, max_workers=threads_per_process
        )
    )

    add_PredictorServicer_to_server = get_servicer_to_server_from_module(module_proto_pb2_grpc)
    add_PredictorServicer_to_server(PredictorServicer(config), server)
    server.add_insecure_port(address)
    server.start()

    time_to_wait = 5.0
    start_time = time.time()
    while time.time() - start_time < time_to_wait:
        time.sleep(1.0)

    open(f"/mnt/workspace/proc-{os.getpid()}-ready.txt", "a").close()
    server.wait_for_termination()


if __name__ == "__main__":
    init_sentry(tags=get_default_tags())
    logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])
    main()
