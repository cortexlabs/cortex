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
import threading
import time
import uuid
from typing import Dict, Any

import boto3
import datadog

from cortex_internal.lib.api import BatchAPI
from cortex_internal.lib.concurrency import LockedFile
from cortex_internal.lib.exceptions import UserRuntimeException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.metrics import MetricsClient
from cortex_internal.lib.queue.sqs import SQSHandler, get_total_messages_in_queue
from cortex_internal.lib.telemetry import get_default_tags, init_sentry, capture_exception

init_sentry(tags=get_default_tags())
log = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])

SQS_POLL_WAIT_TIME = 10  # seconds
MESSAGE_NOT_FOUND_SLEEP = 10  # seconds
INITIAL_MESSAGE_VISIBILITY = 30  # seconds
MESSAGE_RENEWAL_PERIOD = 15  # seconds
JOB_COMPLETE_MESSAGE_RENEWAL = 10  # seconds

local_cache: Dict[str, Any] = {
    "api": None,
    "job_spec": None,
    "handler_impl": None,
    "sqs_client": None,
}

receipt_handle_mutex = threading.Lock()
stop_renewal = set()


def dimensions():
    return [
        {"Name": "api_name", "Value": local_cache["api"].name},
        {"Name": "job_id", "Value": local_cache["job_spec"]["job_id"]},
    ]


def success_counter_metric():
    return {
        "MetricName": "cortex_batch_succeeded",
        "Dimensions": dimensions(),
        "Unit": "Count",
        "Value": 1,
    }


def failed_counter_metric():
    return {
        "MetricName": "cortex_batch_failed",
        "Dimensions": dimensions(),
        "Unit": "Count",
        "Value": 1,
    }


def time_per_batch_metric(total_time_seconds):
    return {
        "MetricName": "cortex_time_per_batch",
        "Dimensions": dimensions(),
        "Value": total_time_seconds,
    }


def build_handle_batch_args(payload, batch_id):
    args = {}

    if "payload" in local_cache["handle_batch_fn_args"]:
        args["payload"] = payload
    if "headers" in local_cache["handle_batch_fn_args"]:
        args["headers"] = None
    if "query_params" in local_cache["handle_batch_fn_args"]:
        args["query_params"] = None
    if "batch_id" in local_cache["handle_batch_fn_args"]:
        args["batch_id"] = batch_id
    return args


def handle_batch_message(message):
    handler_impl = local_cache["handler_impl"]
    api: BatchAPI = local_cache["api"]

    log.info(f"processing batch {message['MessageId']}")

    start_time = time.time()
    payload = json.loads(message["Body"])
    batch_id = message["MessageId"]

    try:
        handler_impl.handle_batch(**build_handle_batch_args(payload, batch_id))
    except Exception as err:
        raise UserRuntimeException from err

    api.metrics.post_metrics(
        [success_counter_metric(), time_per_batch_metric(time.time() - start_time)]
    )


def handle_batch_failure(message):
    api: BatchAPI = local_cache["api"]

    api.metrics.post_metrics([failed_counter_metric()])
    log.exception(f"failed processing batch {message['MessageId']}")


def on_job_complete(message):
    job_spec = local_cache["job_spec"]
    handler_impl = local_cache["handler_impl"]
    sqs_client = local_cache["sqs_client"]
    queue_url = job_spec["sqs_url"]

    should_run_on_job_complete = False

    while True:
        visible_messages, invisible_messages = get_total_messages_in_queue(
            sqs_client=sqs_client, queue_url=queue_url
        )
        total_messages = visible_messages + invisible_messages
        if total_messages > 1:
            new_message_id = uuid.uuid4()
            time.sleep(JOB_COMPLETE_MESSAGE_RENEWAL)
            sqs_client.send_message(
                QueueUrl=queue_url,
                MessageBody='"job_complete"',
                MessageAttributes={
                    "job_complete": {"StringValue": "true", "DataType": "String"},
                    "api_name": {"StringValue": job_spec["api_name"], "DataType": "String"},
                    "job_id": {"StringValue": job_spec["job_id"], "DataType": "String"},
                },
                MessageDeduplicationId=str(new_message_id),
                MessageGroupId=str(new_message_id),
            )
            break
        else:
            if should_run_on_job_complete:
                if getattr(handler_impl, "on_job_complete", None):
                    log.info("executing on_job_complete")
                    try:
                        handler_impl.on_job_complete()
                    except Exception as err:
                        raise UserRuntimeException from err
                break
            should_run_on_job_complete = True

        time.sleep(10)  # verify that the queue is empty one more time


def start():
    while not pathlib.Path("/mnt/workspace/init_script_run.txt").is_file():
        time.sleep(0.2)

    api_spec_path = os.environ["CORTEX_API_SPEC"]
    job_spec_path = os.environ["CORTEX_JOB_SPEC"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]

    model_dir = os.getenv("CORTEX_MODEL_DIR")
    host_ip = os.environ["HOST_IP"]
    tf_serving_port = os.getenv("CORTEX_TF_BASE_SERVING_PORT", "9000")
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")

    region = os.getenv("AWS_DEFAULT_REGION")

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

    with open(api_spec_path) as json_file:
        api_spec = json.load(json_file)
    api = BatchAPI(api_spec, statsd_client, model_dir)

    with open(job_spec_path) as json_file:
        job_spec = json.load(json_file)

    sqs_client = boto3.client("sqs", region_name=region)

    client = api.initialize_client(tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port)

    try:
        log.info("loading the handler from {}".format(api.path))
        handler_impl = api.initialize_impl(
            project_dir=project_dir,
            client=client,
            metrics_client=MetricsClient(statsd_client),
            job_spec=job_spec,
        )
    except UserRuntimeException as err:
        err.wrap(f"failed to start job {job_spec['job_id']}")
        log.error(str(err), exc_info=True)
        sys.exit(1)
    except Exception as err:
        capture_exception(err)
        log.error(f"failed to start job {job_spec['job_id']}", exc_info=True)
        sys.exit(1)

    local_cache["api"] = api
    local_cache["job_spec"] = job_spec
    local_cache["handler_impl"] = handler_impl
    local_cache["handle_batch_fn_args"] = inspect.getfullargspec(handler_impl.handle_batch).args
    local_cache["sqs_client"] = sqs_client

    open("/mnt/workspace/api_readiness.txt", "a").close()

    log.info("polling for batches...")
    try:
        sqs_handler = SQSHandler(
            sqs_client=sqs_client,
            queue_url=job_spec["sqs_url"],
            renewal_period=MESSAGE_RENEWAL_PERIOD,
            visibility_timeout=INITIAL_MESSAGE_VISIBILITY,
            not_found_sleep_time=MESSAGE_NOT_FOUND_SLEEP,
            message_wait_time=SQS_POLL_WAIT_TIME,
            dead_letter_queue_url=job_spec.get("sqs_dead_letter_queue"),
            stop_if_no_messages=True,
        )
        sqs_handler.start(
            message_fn=handle_batch_message,
            message_failure_fn=handle_batch_failure,
            on_job_complete_fn=on_job_complete,
        )
    except UserRuntimeException as err:
        err.wrap(f"failed to run job {job_spec['job_id']}")
        log.error(str(err), exc_info=True)
        sys.exit(1)
    except Exception as err:
        capture_exception(err)
        log.error(f"failed to run job {job_spec['job_id']}", exc_info=True)
        sys.exit(1)


if __name__ == "__main__":
    start()
