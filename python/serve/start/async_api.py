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
import inspect
import json
import os
import pathlib
import sys
import time
from typing import Dict, Any

import boto3

from cortex_internal.lib.api.async_api import AsyncAPI
from cortex_internal.lib.exceptions import UserRuntimeException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.metrics import MetricsClient
from cortex_internal.lib.queue.sqs import SQSHandler
from cortex_internal.lib.storage import S3
from cortex_internal.lib.telemetry import init_sentry, get_default_tags, capture_exception

init_sentry(tags=get_default_tags())
log = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])

SQS_POLL_WAIT_TIME = 10  # seconds
MESSAGE_NOT_FOUND_SLEEP = 10  # seconds
INITIAL_MESSAGE_VISIBILITY = 30  # seconds
MESSAGE_RENEWAL_PERIOD = 15  # seconds
JOB_COMPLETE_MESSAGE_RENEWAL = 10  # seconds

local_cache: Dict[str, Any] = {
    "api": None,
    "handler_impl": None,
    "handle_async_fn_args": None,
    "sqs_client": None,
    "storage_client": None,
}


def handle_workload(message):
    api: AsyncAPI = local_cache["api"]
    handler_impl = local_cache["handler_impl"]

    request_id = message["Body"]
    log.info(f"processing workload...", extra={"id": request_id})

    api.update_status(request_id, "in_progress")
    payload = api.get_payload(request_id)

    try:
        result = handler_impl.handle_async(**build_handle_async_args(payload, request_id))
    except Exception as err:
        raise UserRuntimeException from err

    log.debug("uploading result", extra={"id": request_id})
    api.upload_result(request_id, result)

    log.debug("updating status to completed", extra={"id": request_id})
    api.update_status(request_id, "completed")

    log.debug("deleting payload from s3")
    api.delete_payload(request_id=request_id)

    log.info("workload processing complete", extra={"id": request_id})


def handle_workload_failure(message):
    api: AsyncAPI = local_cache["api"]
    request_id = message["Body"]

    log.error("failed to process workload", exc_info=True, extra={"id": request_id})
    api.update_status(request_id, "failed")

    log.debug("deleting payload from s3")
    api.delete_payload(request_id=request_id)


def build_handle_async_args(payload, request_id):
    args = {}
    if "payload" in local_cache["handle_async_fn_args"]:
        args["payload"] = payload
    if "request_id" in local_cache["handle_async_fn_args"]:
        args["request_id"] = request_id
    return args


def main():
    while not pathlib.Path("/mnt/workspace/init_script_run.txt").is_file():
        time.sleep(0.2)

    model_dir = os.getenv("CORTEX_MODEL_DIR")
    api_spec_path = os.environ["CORTEX_API_SPEC"]
    workload_path = os.environ["CORTEX_ASYNC_WORKLOAD_PATH"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]
    readiness_file = os.getenv("CORTEX_READINESS_FILE", "/mnt/workspace/api_readiness.txt")
    region = os.getenv("AWS_DEFAULT_REGION")
    queue_url = os.environ["CORTEX_QUEUE_URL"]
    statsd_host = os.getenv("HOST_IP")
    statsd_port = os.getenv("CORTEX_STATSD_PORT", "9125")
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST")
    tf_serving_port = os.getenv("CORTEX_TF_BASE_SERVING_PORT")

    bucket, key = S3.deconstruct_s3_path(workload_path)
    storage = S3(bucket=bucket, region=region)

    with open(api_spec_path) as json_file:
        api_spec = json.load(json_file)

    sqs_client = boto3.client("sqs", region_name=region)
    api = AsyncAPI(
        api_spec=api_spec,
        storage=storage,
        storage_path=key,
        statsd_host=statsd_host,
        statsd_port=int(statsd_port),
        model_dir=model_dir,
    )

    try:
        log.info(f"loading the handler from {api.path}")
        metrics_client = MetricsClient(api.statsd)
        handler_impl = api.initialize_impl(
            project_dir,
            metrics_client,
            tf_serving_host=tf_serving_host,
            tf_serving_port=tf_serving_port,
        )
    except UserRuntimeException as err:
        err.wrap(f"failed to initialize handler implementation")
        log.error(str(err), exc_info=True)
        sys.exit(1)
    except Exception as err:
        capture_exception(err)
        log.error(f"failed to initialize handler implementation", exc_info=True)
        sys.exit(1)

    local_cache["api"] = api
    local_cache["handler_impl"] = handler_impl
    local_cache["sqs_client"] = sqs_client
    local_cache["storage_client"] = storage
    local_cache["handle_async_fn_args"] = inspect.getfullargspec(handler_impl.handle_async).args

    open(readiness_file, "a").close()

    log.info("polling for workloads...")
    try:
        sqs_handler = SQSHandler(
            sqs_client=sqs_client,
            queue_url=queue_url,
            renewal_period=MESSAGE_RENEWAL_PERIOD,
            visibility_timeout=INITIAL_MESSAGE_VISIBILITY,
            not_found_sleep_time=MESSAGE_NOT_FOUND_SLEEP,
            message_wait_time=SQS_POLL_WAIT_TIME,
        )
        sqs_handler.start(message_fn=handle_workload, message_failure_fn=handle_workload_failure)
    except UserRuntimeException as err:
        log.error(str(err), exc_info=True)
        sys.exit(1)
    except Exception as err:
        capture_exception(err)
        log.error(str(err), exc_info=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
