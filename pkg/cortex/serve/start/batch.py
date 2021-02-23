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
import sys
import signal
import threading
import pathlib
import time
import uuid
from typing import Dict, Any

import boto3
import botocore
from cortex_internal.lib.api import get_api, get_spec
from cortex_internal.lib.concurrency import LockedFile
from cortex_internal.lib.storage import S3
from cortex_internal.lib.telemetry import get_default_tags, init_sentry, capture_exception
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.exceptions import UserException, UserRuntimeException

init_sentry(tags=get_default_tags())
logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])

SQS_POLL_WAIT_TIME = 10  # seconds
MESSAGE_NOT_FOUND_SLEEP = 10  # seconds
INITIAL_MESSAGE_VISIBILITY = 30  # seconds
MESSAGE_RENEWAL_PERIOD = 15  # seconds
JOB_COMPLETE_MESSAGE_RENEWAL = 10  # seconds

local_cache: Dict[str, Any] = {
    "api": None,
    "job_spec": None,
    "provider": None,
    "predictor_impl": None,
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


def renew_message_visibility(receipt_handle: str):
    queue_url = local_cache["job_spec"]["sqs_url"]
    interval = MESSAGE_RENEWAL_PERIOD
    new_timeout = INITIAL_MESSAGE_VISIBILITY
    cur_time = time.time()

    while True:
        time.sleep((cur_time + interval) - time.time())
        cur_time += interval
        new_timeout += interval

        with receipt_handle_mutex:
            if receipt_handle in stop_renewal:
                stop_renewal.remove(receipt_handle)
                break

            try:
                local_cache["sqs_client"].change_message_visibility(
                    QueueUrl=queue_url, ReceiptHandle=receipt_handle, VisibilityTimeout=new_timeout
                )
            except botocore.exceptions.ClientError as err:
                if err.response["Error"]["Code"] == "InvalidParameterValue":
                    # unexpected; this error is thrown when attempting to renew a message that has been deleted
                    continue
                elif err.response["Error"]["Code"] == "AWS.SimpleQueueService.NonExistentQueue":
                    # there may be a delay between the cron may deleting the queue and this worker stopping
                    logger.info(
                        "failed to renew message visibility because the queue was not found"
                    )
                else:
                    stop_renewal.remove(receipt_handle)
                    raise err


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
    sqs_client = local_cache["sqs_client"]

    queue_url = job_spec["sqs_url"]

    no_messages_found_in_previous_iteration = False

    while True:
        response = sqs_client.receive_message(
            QueueUrl=queue_url,
            MaxNumberOfMessages=1,
            WaitTimeSeconds=SQS_POLL_WAIT_TIME,
            VisibilityTimeout=INITIAL_MESSAGE_VISIBILITY,
            MessageAttributeNames=["All"],
        )

        if response.get("Messages") is None or len(response["Messages"]) == 0:
            visible_messages, invisible_messages = get_total_messages_in_queue()
            if visible_messages + invisible_messages == 0:
                if no_messages_found_in_previous_iteration:
                    logger.info("no batches left in queue, exiting...")
                    return
                no_messages_found_in_previous_iteration = True

            time.sleep(MESSAGE_NOT_FOUND_SLEEP)
            continue

        no_messages_found_in_previous_iteration = False
        message = response["Messages"][0]
        receipt_handle = message["ReceiptHandle"]

        renewer = threading.Thread(
            target=renew_message_visibility, args=(receipt_handle,), daemon=True
        )
        renewer.start()

        if is_on_job_complete(message):
            handle_on_job_complete(message)
        else:
            handle_batch_message(message)


def is_on_job_complete(message) -> bool:
    return "MessageAttributes" in message and "job_complete" in message["MessageAttributes"]


def handle_batch_message(message):
    job_spec = local_cache["job_spec"]
    predictor_impl = local_cache["predictor_impl"]
    sqs_client = local_cache["sqs_client"]
    queue_url = job_spec["sqs_url"]
    receipt_handle = message["ReceiptHandle"]
    api = local_cache["api"]

    start_time = time.time()

    try:
        logger.info(f"processing batch {message['MessageId']}")
        payload = json.loads(message["Body"])
        batch_id = message["MessageId"]

        try:
            predictor_impl.predict(**build_predict_args(payload, batch_id))
        except Exception as err:
            raise UserRuntimeException from err

        api.post_metrics(
            [success_counter_metric(), time_per_batch_metric(time.time() - start_time)]
        )
    except (UserRuntimeException, Exception) as err:
        if not isinstance(err, UserRuntimeException):
            capture_exception(err)

        api.post_metrics([failed_counter_metric()])
        logger.exception(f"failed processing batch {message['MessageId']}")
        with receipt_handle_mutex:
            stop_renewal.add(receipt_handle)
            if job_spec.get("sqs_dead_letter_queue") is not None:
                sqs_client.change_message_visibility(  # return message
                    QueueUrl=queue_url, ReceiptHandle=receipt_handle, VisibilityTimeout=0
                )
            else:
                sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)
    else:
        with receipt_handle_mutex:
            stop_renewal.add(receipt_handle)
            sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)


def handle_on_job_complete(message):
    job_spec = local_cache["job_spec"]
    predictor_impl = local_cache["predictor_impl"]
    sqs_client = local_cache["sqs_client"]
    queue_url = job_spec["sqs_url"]
    receipt_handle = message["ReceiptHandle"]

    should_run_on_job_complete = False
    try:
        while True:
            visible_messages, invisible_messages = get_total_messages_in_queue()
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
                    if getattr(predictor_impl, "on_job_complete", None):
                        logger.info("executing on_job_complete")
                        try:
                            predictor_impl.on_job_complete()
                        except Exception as err:
                            raise UserRuntimeException from err
                    break
                should_run_on_job_complete = True
            time.sleep(10)  # verify that the queue is empty one more time
    except (UserRuntimeException, Exception) as err:
        raise type(err)("failed to handle on_job_complete") from err
    finally:
        with receipt_handle_mutex:
            stop_renewal.add(receipt_handle)
            sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)


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
    storage, _ = get_spec(provider, api_spec_path, cache_dir, region)
    job_spec = get_job_spec(storage, cache_dir, job_spec_path)

    client = api.predictor.initialize_client(
        tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
    )

    try:
        logger.info("loading the predictor from {}".format(api.predictor.path))
        predictor_impl = api.predictor.initialize_impl(project_dir, client, job_spec)
    except (UserException, UserRuntimeException) as err:
        err.wrap(f"failed to start job {job_spec['job_id']}")
        logger.error(str(err), exc_info=True)
        sys.exit(1)
    except Exception as err:
        capture_exception(err)
        logger.error(f"failed to start job {job_spec['job_id']}", exc_info=True)
        sys.exit(1)

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
    local_cache["job_spec"] = job_spec
    local_cache["predictor_impl"] = predictor_impl
    local_cache["predict_fn_args"] = inspect.getfullargspec(predictor_impl.predict).args
    local_cache["sqs_client"] = boto3.client("sqs", region_name=region)

    open("/mnt/workspace/api_readiness.txt", "a").close()

    logger.info("polling for batches...")
    try:
        sqs_loop()
    except UserRuntimeException as err:
        err.wrap(f"failed to run job {job_spec['job_id']}")
        logger.error(str(err), exc_info=True)
        sys.exit(1)
    except Exception as err:
        capture_exception(err)
        logger.error(f"failed to run job {job_spec['job_id']}", exc_info=True)
        sys.exit(1)


if __name__ == "__main__":
    start()
