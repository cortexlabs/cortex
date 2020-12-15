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
import threading
import math
import pathlib
import uuid

import boto3
import botocore

from cortex import consts
from cortex.lib import util
from cortex.lib.api import API, get_spec, get_api
from cortex.lib.log import cx_logger as logger
from cortex.lib.concurrency import LockedFile
from cortex.lib.storage import S3, LocalStorage
from cortex.lib.exceptions import UserRuntimeException

from concurrent.futures import ThreadPoolExecutor
import asyncio
from typing import Any

API_LIVENESS_UPDATE_PERIOD = 5  # seconds
SQS_POLL_WAIT_TIME = 10  # seconds
MESSAGE_NOT_FOUND_SLEEP = 10  # seconds
INITIAL_MESSAGE_VISIBILITY = 60  # seconds
MESSAGE_RENEWAL_PERIOD = 30  # seconds

local_cache = {
    "api_spec": None,
    "job_spec": None,
    "provider": None,
    "predictor_impl": None,
    "predict_route": None,
    "client": None,
    "class_set": set(),
    "sqs_client": None,
}


def dimensions():
    return [
        {"Name": "APIName", "Value": local_cache["api_spec"].name},
        {"Name": "JobID", "Value": local_cache["job_spec"]["job_id"]},
    ]


def success_counter_metric():
    return {"MetricName": "Succeeded", "Dimensions": dimensions(), "Unit": "Count", "Value": 1}


def failed_counter_metric():
    return {"MetricName": "Failed", "Dimensions": dimensions(), "Unit": "Count", "Value": 1}


def time_per_batch_metric(total_time_seconds):
    return {"MetricName": "TimePerBatch", "Dimensions": dimensions(), "Value": total_time_seconds}


def update_api_liveness():
    threading.Timer(API_LIVENESS_UPDATE_PERIOD, update_api_liveness).start()
    with open("/mnt/workspace/api_liveness.txt", "w") as f:
        f.write(str(math.ceil(time.time())))


# TODO Take a look at usage scenario
def startup():
    open("/mnt/workspace/api_readiness.txt", "a").close()
    update_api_liveness()


stop_renewal = set()


def renew_message_visibility(receipt_handle: str):
    queue_url = local_cache["job_spec"]["sqs_url"]
    interval = MESSAGE_RENEWAL_PERIOD
    new_timeout = INITIAL_MESSAGE_VISIBILITY
    cur_time = time.time()

    while True:
        time.sleep((cur_time + interval) - time.time())
        print("executing renew_message_visibility")
        cur_time += interval
        new_timeout += interval

        if receipt_handle in stop_renewal:
            stop_renewal.remove(receipt_handle)
            break

        try:
            local_cache["sqs_client"].change_message_visibility(
                QueueUrl=queue_url, ReceiptHandle=receipt_handle, VisibilityTimeout=new_timeout
            )
        except botocore.exceptions.ClientError as e:
            if e.response["Error"]["Code"] == "InvalidParameterValue":
                continue
            elif e.response["Error"]["Code"] == "AWS.SimpleQueueService.NonExistentQueue":
                cx_logger().info(
                    "failed to renew message visibility because the queue was not found"
                )
            else:
                stop_renewal.remove(receipt_handle)
                raise e

        new_timeout += interval


def build_predict_args(payload, batch_id):
    args = {}

    if "payload" in local_cache["predict_fn_args"]:
        args["payload"] = payload
    if "headers" in local_cache["predict_fn_args"]:
        args["headers"] = None
    if "query_params" in local_cache["predict_fn_args"]:
        args["query_params"] = None
    if "batch_id" in local_cache["predict_fn_args"]:
        args["batch_id"] = batch_id
    return args


def get_job_spec(storage, cache_dir, job_spec_path):
    local_spec_path = os.path.join(cache_dir, "job_spec.json")
    _, key = S3.deconstruct_s3_path(job_spec_path)
    storage.download_file(key, local_spec_path)
    with open(local_spec_path) as f:
        return json.load(f)


def get_total_messages_in_queue():
    sqs_client = local_cache["sqs_client"]
    job_spec = local_cache["job_spec"]
    queue_url = job_spec["sqs_url"]

    attributes = sqs_client.get_queue_attributes(QueueUrl=queue_url, AttributeNames=["All"])[
        "Attributes"
    ]
    visible_count = int(attributes.get("ApproximateNumberOfMessages", 0))
    not_visible_count = int(attributes.get("ApproximateNumberOfMessagesNotVisible", 0))
    return visible_count, not_visible_count


def sqs_loop():
    job_spec = local_cache["job_spec"]
    api_spec = local_cache["api_spec"]
    predictor_impl = local_cache["predictor_impl"]
    sqs_client = local_cache["sqs_client"]

    queue_url = job_spec["sqs_url"]

    no_messages_found_in_previous_iteration = False

    while True:
        response = sqs_client.receive_message(
            QueueUrl=queue_url,
            MaxNumberOfMessages=1,
            WaitTimeSeconds=10,
            VisibilityTimeout=INITIAL_MESSAGE_VISIBILITY,
            # VisibilityTimeout=MAXIMUM_MESSAGE_VISIBILITY,
            MessageAttributeNames=["All"],
        )

        if response.get("Messages") is None or len(response["Messages"]) == 0:
            visible_messages, invisible_messages = get_total_messages_in_queue()
            if visible_messages + invisible_messages == 0:
                if no_messages_found_in_previous_iteration:
                    logger().info("no batches left in queue, exiting...")
                    return
                no_messages_found_in_previous_iteration = True

            time.sleep(MESSAGE_NOT_FOUND_SLEEP)
            continue

        no_messages_found_in_previous_iteration = False
        message = response["Messages"][0]
        receipt_handle = message["ReceiptHandle"]

        try:
            renewer = threading.Thread(
                target=renew_message_visibility, args=(receipt_handle,), daemon=True
            )
            renewer.start()
            start_time = time.time()

            if is_on_job_complete(message):
                handle_on_job_complete(message)
            else:
                logger().info(f"processing batch {message['MessageId']}")
                payload = json.loads(message["Body"])
                batch_id = message["MessageId"]
                predictor_impl.predict(**build_predict_args(payload, batch_id))

                api_spec.post_metrics(
                    [success_counter_metric(), time_per_batch_metric(time.time() - start_time)]
                )
        except Exception:
            if is_on_job_complete(message):
                logger().exception("failed to execute on_job_complete")
            else:
                api_spec.post_metrics(
                    [failed_counter_metric(), time_per_batch_metric(time.time() - start_time)]
                )
                logger().exception("failed to process batch")
        finally:
            stop_renewal.add(receipt_handle)
            sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)


def is_on_job_complete(message) -> bool:
    return "MessageAttributes" in message and "job_complete" in message["MessageAttributes"]


def handle_on_job_complete(message):
    job_spec = local_cache["job_spec"]
    predictor_impl = local_cache["predictor_impl"]
    sqs_client = local_cache["sqs_client"]
    queue_url = job_spec["sqs_url"]

    should_run_on_job_complete = False

    while True:
        visible_messages, invisible_messages = get_total_messages_in_queue()
        total_messages = visible_messages + invisible_messages
        if total_messages > 1:
            new_message_id = uuid.uuid4()
            sqs_client.send_message(
                QueueUrl=queue_url,
                MessageBody='"job_complete"',
                MessageAttributes={"job_complete": {"StringValue": "true", "DataType": "string"}},
                MessageDeduplicationId=new_message_id,
                MessageGroupId=new_message_id,
            )
            return
        else:
            if should_run_on_job_complete:
                if getattr(predictor_impl, "on_job_complete", None):
                    logger().info("executing on_job_complete")
                    predictor_impl.on_job_complete()
                    return
            should_run_on_job_complete = True
        time.sleep(10)  # verify that the queue is empty one more time


def start():
    while not pathlib.Path("/mnt/workspace/init_script_run.txt").is_file():
        time.sleep(0.2)

    cache_dir = os.environ["CORTEX_CACHE_DIR"]
    provider = os.environ["CORTEX_PROVIDER"]
    api_spec_path = os.environ["CORTEX_API_SPEC"]
    job_spec_path = os.environ["CORTEX_JOB_SPEC"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]

    model_dir = os.getenv("CORTEX_MODEL_DIR")
    tf_serving_port = os.getenv("CORTEX_TF_BASE_SERVING_PORT", "9000")
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")

    region = os.getenv("AWS_REGION")

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

    api = get_api(provider, api_spec_path, model_dir, cache_dir, region)
    storage, api_spec = get_spec(provider, api_spec_path, cache_dir, region)
    job_spec = get_job_spec(storage, cache_dir, job_spec_path)

    client = api.predictor.initialize_client(
        tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
    )
    logger().info("loading the predictor from {}".format(api.predictor.path))
    predictor_impl = api.predictor.initialize_impl(project_dir, client, job_spec)

    local_cache["api_spec"] = api
    local_cache["provider"] = provider
    local_cache["job_spec"] = job_spec
    local_cache["predictor_impl"] = predictor_impl
    local_cache["predict_fn_args"] = inspect.getfullargspec(predictor_impl.predict).args
    local_cache["sqs_client"] = boto3.client("sqs", region_name=region)

    open("/mnt/workspace/api_readiness.txt", "a").close()

    logger().info("polling for batches...")
    sqs_loop()
    logger().info("done")


if __name__ == "__main__":
    start()
    logger().info("completed main")
