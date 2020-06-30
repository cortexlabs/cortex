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

import uuid
import boto3

from cortex import consts
from cortex.lib import util
from cortex.lib.type import API, get_spec
from cortex.lib.log import cx_logger
from cortex.lib.storage import S3, LocalStorage, FileLock
from cortex.lib.exceptions import UserRuntimeException

if os.environ["CORTEX_VERSION"] != consts.CORTEX_VERSION:
    errMsg = f"your Cortex operator version ({os.environ['CORTEX_VERSION']}) doesn't match your predictor image version ({consts.CORTEX_VERSION}); please update your predictor image by modifying the `image` field in your API configuration file (e.g. cortex.yaml) and re-running `cortex deploy`, or update your cluster by following the instructions at https://docs.cortex.dev/cluster-management/update"
    raise ValueError(errMsg)


API_LIVENESS_UPDATE_PERIOD = 5  # seconds


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


class Ticker:
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


def time_per_partition_metric(total_time_seconds):
    return {
        "MetricName": "TimePerPartition",
        "Dimensions": dimensions(),
        "Value": total_time_seconds,
    }


def update_api_liveness():
    threading.Timer(API_LIVENESS_UPDATE_PERIOD, update_api_liveness).start()
    with open("/mnt/workspace/api_liveness.txt", "w") as f:
        f.write(str(math.ceil(time.time())))


def startup():
    open("/mnt/workspace/api_readiness.txt", "a").close()
    update_api_liveness()


def renew_message_visibility(queue_url, receipt_handle, initial_offset, interval, *args, **kwargs):
    new_timeout = initial_offset + interval
    while True:
        yield
        print("resetting")
        output = local_cache["sqs"].change_message_visibility(
            QueueUrl=queue_url, ReceiptHandle=receipt_handle, VisibilityTimeout=new_timeout
        )
        print(output)
        new_timeout += interval


def get_api_spec(provider, storage, cache_dir, api_spec_path):
    if provider == "local":
        return read_msgpack(api_spec_path)

    local_spec_path = os.path.join(cache_dir, "api_spec.msgpack")
    _, key = S3.deconstruct_s3_path(api_spec_path)
    storage.download_file(key, local_spec_path)
    return read_msgpack(local_spec_path)


def read_msgpack(msgpack_path):
    with open(msgpack_path, "rb") as msgpack_file:
        return msgpack.load(msgpack_file, raw=False)


def get_job_spec(storage, cache_dir, job_spec_path):
    local_spec_path = os.path.join(cache_dir, "job_spec.json")
    _, key = S3.deconstruct_s3_path(job_spec_path)
    storage.download_file(key, local_spec_path)
    with open(local_spec_path) as f:
        return json.load(f)


def sqs_loop():
    queue_url = os.environ.get("SQS_QUEUE_URL")

    open("/mnt/workspace/api_readiness.txt", "a").close()

    print(queue_url)

    while True:
        print("entering loop")
        request_attempt = str(uuid.uuid4())
        response = local_cache["sqs"].receive_message(
            QueueUrl=queue_url,
            AttributeNames=["SentTimestamp"],
            MaxNumberOfMessages=1,
            MessageAttributeNames=["All"],
            WaitTimeSeconds=1,
            VisibilityTimeout=120,
            ReceiveRequestAttemptId=request_attempt,
        )

        print(response)
        if response.get("Messages") is None or len(response["Messages"]) == 0:
            print("no results")
            break

        start_time = time.time()

        renewer = Ticker(
            60,
            renew_message_visibility,
            queue_url,
            response["Messages"][0]["ReceiptHandle"],
            120,
            60,
        )

        renewer.start()
        try:
            local_cache["predictor_impl"].predict(payload=response["Messages"][0]["Body"])
            local_cache["api"].post_metrics(
                [success_counter_metric(), time_per_partition_metric(time.time() - start_time)]
            )
        except Exception as e:
            local_cache["api"].post_metrics(
                [failed_counter_metric(), time_per_partition_metric(time.time() - start_time)]
            )
        finally:
            renewer.stop()
            local_cache["sqs"].delete_message(
                QueueUrl=queue_url, ReceiptHandle=response["Messages"][0]["ReceiptHandle"]
            )


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
        print(job_spec)
        if job_spec.get("config") is not None:
            raw_api_spec["predictor"]["config"] = util.merge_dicts_overwrite(
                raw_api_spec["predictor"]["config"], job_spec["config"]
            )

        client = api.predictor.initialize_client(
            tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
        )
        cx_logger().info("loading the predictor from {}".format(api.predictor.path))
        predictor_impl = api.predictor.initialize_impl(project_dir, client)

        local_cache["api"] = api
        local_cache["provider"] = provider
        local_cache["client"] = client
        local_cache["job"] = job_spec
        local_cache["predictor_impl"] = predictor_impl
        local_cache["predict_fn_args"] = inspect.getfullargspec(predictor_impl.predict).args
        local_cache["sqs"] = boto3.client("sqs", region_name=os.environ["AWS_REGION"])
    except:
        cx_logger().exception("failed to start api")
        sys.exit(1)

    print("starting sqs loop...")
    sqs_loop()


if __name__ == "__main__":
    start()

# 69e2c3a88a3267b4
