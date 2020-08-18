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
import msgpack
import threading
import math

import boto3
import botocore

from cortex import consts
from cortex.lib import util
from cortex.lib.type import API, get_spec
from cortex.lib.log import cx_logger
from cortex.lib.storage import S3, LocalStorage, FileLock
from cortex.lib.exceptions import UserRuntimeException

API_LIVENESS_UPDATE_PERIOD = 5  # seconds
MAXIMUM_MESSAGE_VISIBILITY = 60 * 60 * 12  # 12 hours is the maximum message visibility

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


def handle_on_complete(message):
    job_spec = local_cache["job_spec"]
    predictor_impl = local_cache["predictor_impl"]
    sqs_client = local_cache["sqs_client"]
    queue_url = job_spec["sqs_url"]
    receipt_handle = message["ReceiptHandle"]

    try:
        if not getattr(predictor_impl, "on_job_complete", None):
            sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)
            return True

        should_run_on_job_complete = False

        while True:
            visible_count, not_visible_count = get_total_messages_in_queue()

            # if there are other messages that are visible, release this message and get the other ones (should rarely happen for FIFO)
            if visible_count > 0:
                sqs_client.change_message_visibility(
                    QueueUrl=queue_url, ReceiptHandle=receipt_handle, VisibilityTimeout=0
                )
                return False

            if should_run_on_job_complete:
                # double check that the queue is still empty (except for the job_complete message)
                if not_visible_count <= 1:
                    cx_logger().info("executing on_job_complete")
                    predictor_impl.on_job_complete()
                    sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)
                    return True
                else:
                    should_run_on_job_complete = False

            if not_visible_count <= 1:
                should_run_on_job_complete = True

            time.sleep(20)
    except:
        sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)
        raise


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
            VisibilityTimeout=MAXIMUM_MESSAGE_VISIBILITY,
            MessageAttributeNames=["All"],
        )

        if response.get("Messages") is None or len(response["Messages"]) == 0:
            if no_messages_found_in_previous_iteration:
                cx_logger().info("no batches left in queue, exiting...")
                return
            else:
                no_messages_found_in_previous_iteration = True
                continue
        else:
            no_messages_found_in_previous_iteration = False

        message = response["Messages"][0]

        receipt_handle = message["ReceiptHandle"]

        if "MessageAttributes" in message and "job_complete" in message["MessageAttributes"]:
            handled_on_complete = handle_on_complete(message)
            if handled_on_complete:
                cx_logger().info("no batches left in queue, job has been completed")
                return
            else:
                # sometimes on_job_complete message will be released if there are other messages still to be processed
                continue

        try:
            cx_logger().info(f"processing batch {message['MessageId']}")

            start_time = time.time()

            payload = json.loads(message["Body"])
            batch_id = message["MessageId"]
            predictor_impl.predict(**build_predict_args(payload, batch_id))

            api_spec.post_metrics(
                [success_counter_metric(), time_per_batch_metric(time.time() - start_time)]
            )
        except Exception:
            api_spec.post_metrics(
                [failed_counter_metric(), time_per_batch_metric(time.time() - start_time)]
            )
            cx_logger().exception("failed to process batch")
        finally:
            sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)


def start():
    cache_dir = os.environ["CORTEX_CACHE_DIR"]
    provider = os.environ["CORTEX_PROVIDER"]
    api_spec_path = os.environ["CORTEX_API_SPEC"]
    job_spec_path = os.environ["CORTEX_JOB_SPEC"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]

    model_dir = os.getenv("CORTEX_MODEL_DIR")
    tf_serving_port = os.getenv("CORTEX_TF_BASE_SERVING_PORT", "9000")
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")

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

    raw_api_spec = get_spec(provider, storage, cache_dir, api_spec_path)
    job_spec = get_job_spec(storage, cache_dir, job_spec_path)

    api = API(
        provider=provider, storage=storage, model_dir=model_dir, cache_dir=cache_dir, **raw_api_spec
    )

    client = api.predictor.initialize_client(
        tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
    )
    cx_logger().info("loading the predictor from {}".format(api.predictor.path))
    predictor_impl = api.predictor.initialize_impl(project_dir, client, raw_api_spec, job_spec)

    local_cache["api_spec"] = api
    local_cache["provider"] = provider
    local_cache["job_spec"] = job_spec
    local_cache["predictor_impl"] = predictor_impl
    local_cache["predict_fn_args"] = inspect.getfullargspec(predictor_impl.predict).args
    local_cache["sqs_client"] = boto3.client("sqs", region_name=os.environ["AWS_REGION"])

    open("/mnt/workspace/api_readiness.txt", "a").close()

    cx_logger().info("polling for batches...")
    sqs_loop()


if __name__ == "__main__":
    start()
