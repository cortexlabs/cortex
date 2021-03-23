import os
import sys
import json
import time
import uuid
import signal
import threading
import traceback
from typing import Dict, Any
from concurrent import futures

import grpc

from cortex_internal.lib.api import get_api
from cortex_internal.lib.api.batching import DynamicBatcher
from cortex_internal.lib.concurrency import FileLock, LockedFile
from cortex_internal.lib.exceptions import UserRuntimeException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.metrics import MetricsClient
from cortex_internal.lib.telemetry import capture_exception, get_default_tags, init_sentry

import iris_classifier_pb2
import iris_classifier_pb2_grpc

NANOSECONDS_IN_SECOND = 1e9

class ThreadPoolExecutorWithRequestMonitor:
    def __init__(self, *args, **kwargs):
        self._thread_pool_executor = futures.ThreadPoolExecutor(*args, **kwargs)

    def submit(self, fn, *args, **kwargs):
        request_id = uuid.uuid1()
        file_id = f"/mnt/requests/{request_id}"
        open(file_id, "a").close()

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
            return result

        self._thread_pool_executor.submit(wrapper_fn, *args, **kwargs)

    def map(self, *args, **kwargs):
        return self._thread_pool_executor.map(*args, **kwargs)

    def shutdown(self, *args, **kwargs):
        return self._thread_pool_executor.shutdown(*args, **kwargs)

class PredictorServicer(iris_classifier_pb2_grpc.PredictorServicer):
    def __init__(self, config: Dict[str, Any]):
        self.config = config

    def Predict(self, payload, context):
        try:
            response = self.config["predictor_impl"].predict(payload=payload)
        except Exception:
            logger.error(traceback.format_exc())
            context.abort(grpc.StatusCode.INTERNAL, "internal server error")
        return response


def main():
    address = sys.argv[1]

    threads_per_process = os.environ["CORTEX_THREADS_PER_PROCESS"]

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
            metrics_client = MetricsClient(api.statsd)
            predictor_impl = api.predictor.initialize_impl(
                project_dir=project_dir, client=client, metrics_client=metrics_client
            )

        # crons only stop if an unhandled exception occurs
        def check_if_crons_have_failed():
            while True:
                for cron in api.predictor.crons:
                    if not cron.is_alive():
                        os.kill(os.getpid(), signal.SIGQUIT)
                time.sleep(1)

        threading.Thread(target=check_if_crons_have_failed, daemon=True).start()

        config: Dict[str, Any] = {
            "api": None,
            "provider": None,
            "client": None,
            "predictor_impl": None,
            "dynamic_batcher": None,
        }

        config["api"] = api
        config["provider"] = provider
        config["client"] = client
        config["predictor_impl"] = predictor_impl

        if api.python_server_side_batching_enabled:
            dynamic_batching_config = api.api_spec["predictor"]["server_side_batching"]
            config["dynamic_batcher"] = DynamicBatcher(
                predictor_impl,
                max_batch_size=dynamic_batching_config["max_batch_size"],
                batch_interval=dynamic_batching_config["batch_interval"]
                / NANOSECONDS_IN_SECOND,  # convert nanoseconds to seconds
            )

    except Exception as err:
        if not isinstance(err, UserRuntimeException):
            capture_exception(err)
        logger.exception("failed to start api")
        sys.exit(1)


    server = grpc.server(ThreadPoolExecutorWithRequestMonitor(max_workers=int(1)))
    iris_classifier_pb2_grpc.add_PredictorServicer_to_server(
        PredictorServicer(config), server)
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
