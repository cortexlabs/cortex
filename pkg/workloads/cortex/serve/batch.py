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
from concurrent.futures import ThreadPoolExecutor
import threading
import math
import asyncio
from typing import Any

import boto3
import botocore

from cortex import consts
from cortex.lib import util
from cortex.lib.type import API, get_spec
from cortex.lib.log import cx_logger
from cortex.lib.storage import S3, LocalStorage, FileLock
from cortex.lib.exceptions import UserRuntimeException

API_LIVENESS_UPDATE_PERIOD = 5  # seconds
INITIAL_MESSAGE_VISIBILITY = 120  # seconds
MESSAGE_RENEWAL_PERIOD = 60  # seconds

local_cache = {
    "api": None,
    "job": None,
    "provider": None,
    "predictor_impl": None,
    "predict_route": None,
    "client": None,
    "class_set": set(),
    "sqs": None,
}


class PeriodicGeneratorRunner:
    def __init__(self, interval, func, *args, **kwargs):
        self.interval = interval
        self.func = func
        self.args = args
        self.kwargs = kwargs
        self.generator = self.func(*self.args, **self.kwargs)

    def start(self):
        self.timer = threading.Timer(self.interval, self.start)
        self.timer.start()
        next(self.generator)

    def stop(self):
        self.timer.cancel()


def dimensions():
    return [
        {"Name": "APIName", "Value": local_cache["api"].name},
        {"Name": "JobID", "Value": local_cache["job"]["job_id"]},
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


def startup():
    open("/mnt/workspace/api_readiness.txt", "a").close()
    update_api_liveness()


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


def renew_message_visibility(queue_url, receipt_handle, initial_offset, interval, *args, **kwargs):
    new_timeout = initial_offset + interval
    while True:
        yield
        try:
            local_cache["sqs"].change_message_visibility(
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
                raise e

        new_timeout += interval


def get_api_spec(provider, storage, cache_dir, api_spec_path):
    local_spec_path = os.path.join(cache_dir, "api_spec.msgpack")
    _, key = S3.deconstruct_s3_path(api_spec_path)
    storage.download_file(key, local_spec_path)
    with open(local_spec_path, "rb") as msgpack_file:
        return msgpack.load(msgpack_file, raw=False)


def get_job_spec(storage, cache_dir, job_spec_path):
    local_spec_path = os.path.join(cache_dir, "job_spec.json")
    _, key = S3.deconstruct_s3_path(job_spec_path)
    storage.download_file(key, local_spec_path)
    with open(local_spec_path) as f:
        return json.load(f)


def get_total_messages_in_queue():
    sqs_client = local_cache["sqs"]
    job_spec = local_cache["job"]
    queue_url = job_spec["sqs_url"]

    attributes = sqs_client.get_queue_attributes(QueueUrl=queue_url, AttributeNames=["All"])[
        "Attributes"
    ]
    visible_count = int(attributes.get("ApproximateNumberOfMessages", 0))
    not_visible_count = int(attributes.get("ApproximateNumberOfMessagesNotVisible", 0))
    return visible_count, not_visible_count


def handle_on_complete(message):
    job_spec = local_cache["job"]
    predictor_impl = local_cache["predictor_impl"]
    sqs_client = local_cache["sqs"]
    queue_url = job_spec["sqs_url"]
    receipt_handle = message["ReceiptHandle"]

    total_message_visibility = INITIAL_MESSAGE_VISIBILITY

    try:
        if not getattr(predictor_impl, "on_job_complete", None):
            sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)
            return

        trigger_on_job_complete_next = False
        while True:
            visible_count, not_visible_count = get_total_messages_in_queue()

            # if there are other messages that are visible, release this message and get the other ones (should rarely for FIFO)
            if visible_count > 0:
                sqs_client.change_message_visibility(
                    QueueUrl=queue_url, ReceiptHandle=receipt_handle, VisibilityTimeout=0
                )
                return

            total_in_queue = visible_count + not_visible_count

            if trigger_on_job_complete_next:
                # double check that the queue is still empty (except for the job_complete message)
                if total_in_queue <= 1:
                    predictor_impl.on_job_complete()
                    cx_logger().info("executed on_job_complete()")
                    sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)
                    return
                else:
                    trigger_on_job_complete_next = False

            if total_in_queue == 1:
                trigger_on_job_complete_next = True

            # check queue state every 20 seconds (give time for queue metrics to achieve consistency)
            total_message_visibility += 20
            sqs_client.change_message_visibility(
                QueueUrl=queue_url,
                ReceiptHandle=receipt_handle,
                VisibilityTimeout=total_message_visibility,
            )
            time.sleep(20)
    except:
        sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)
        raise


def sqs_loop():
    job_spec = local_cache["job"]
    api_spec = local_cache["api"]
    predictor_impl = local_cache["predictor_impl"]
    sqs_client = local_cache["sqs"]

    _, results_dir_key = S3.deconstruct_s3_path(local_cache["job"]["results_dir"])
    queue_url = job_spec["sqs_url"]

    open("/mnt/workspace/api_readiness.txt", "a").close()

    no_messages_found_previous_loop = False

    while True:
        response = sqs_client.receive_message(
            QueueUrl=queue_url,
            MaxNumberOfMessages=1,
            WaitTimeSeconds=10,
            VisibilityTimeout=INITIAL_MESSAGE_VISIBILITY,
            MessageAttributeNames=["All"],
        )

        if response.get("Messages") is None or len(response["Messages"]) == 0:
            if no_messages_found_previous_loop:
                cx_logger().info("no batches left in queue, exiting...")
                break
            else:
                no_messages_found_previous_loop = True
                continue
        else:
            no_messages_found_previous_loop = False

        message = response["Messages"][0]

        receipt_handle = message["ReceiptHandle"]

        if "MessageAttributes" in message and "job_complete" in message["MessageAttributes"]:
            handle_on_complete(message)
            continue

        start_time = time.time()

        renewer = PeriodicGeneratorRunner(
            MESSAGE_RENEWAL_PERIOD,
            renew_message_visibility,
            queue_url,
            receipt_handle,
            INITIAL_MESSAGE_VISIBILITY,
            MESSAGE_RENEWAL_PERIOD,
        )

        renewer.start()
        try:
            payload = json.loads(message["Body"])
            batch_id = message["MessageId"]
            prediction = predictor_impl.predict(**build_predict_args(payload, batch_id))

            if prediction is not None:
                key = f"{results_dir_key}/{batch_id}"

                if isinstance(prediction, bytes):
                    api_spec.storage.put_object(prediction, key)
                if isinstance(prediction, str):
                    api_spec.storage.put_str(prediction, key)
                else:
                    try:
                        response = json.dumps(prediction)
                    except Exception as e:
                        raise UserRuntimeException(
                            str(e),
                            "please return an object that is JSON serializable (including its nested fields), a bytes object or a string; alternatively, you can also not return anything and write the output to your desired storage yourself",
                        ) from e
                    api_spec.storage.put_str(response, key + ".json")
                cx_logger.info(
                    f"uploaded results for {batch_id} to s3://{api_spec.storage.bucket}/{key}"
                )

            api_spec.post_metrics(
                [success_counter_metric(), time_per_batch_metric(time.time() - start_time)]
            )
        except Exception:
            cx_logger().exception("failed to process batch")
            api_spec.post_metrics(
                [failed_counter_metric(), time_per_batch_metric(time.time() - start_time)]
            )
        finally:
            renewer.stop()
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

    try:
        raw_api_spec = get_spec(provider, storage, cache_dir, api_spec_path)
        api = API(
            provider=provider,
            storage=storage,
            model_dir=model_dir,
            cache_dir=cache_dir,
            **raw_api_spec,
        )

        job_spec = get_job_spec(storage, cache_dir, job_spec_path)
        if job_spec.get("config") is not None:
            util.merge_dicts_in_place_overwrite(
                raw_api_spec["predictor"]["config"], job_spec["config"]
            )

        client = api.predictor.initialize_client(
            tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
        )
        cx_logger().info("loading the predictor from {}".format(api.predictor.path))
        predictor_impl = api.predictor.initialize_impl(project_dir, client, raw_api_spec, job_spec)

        local_cache["api"] = api
        local_cache["provider"] = provider
        local_cache["job"] = job_spec
        local_cache["predictor_impl"] = predictor_impl
        local_cache["predict_fn_args"] = inspect.getfullargspec(predictor_impl.predict).args
        local_cache["sqs"] = boto3.client("sqs", region_name=os.environ["AWS_REGION"])
    except:
        cx_logger().exception("failed to start api")
        sys.exit(1)

    cx_logger().info("polling for batches...")
    sqs_loop()


if __name__ == "__main__":
    start()
