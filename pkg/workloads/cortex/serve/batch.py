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

from cortex import consts
from cortex.lib import util
from cortex.lib.type import API
from cortex.lib.log import cx_logger
from cortex.lib.storage import S3, LocalStorage
from cortex.lib.exceptions import UserRuntimeException

if os.environ["CORTEX_VERSION"] != consts.CORTEX_VERSION:
    errMsg = f"your Cortex operator version ({os.environ['CORTEX_VERSION']}) doesn't match your predictor image version ({consts.CORTEX_VERSION}); please update your predictor image by modifying the `image` field in your API configuration file (e.g. cortex.yaml) and re-running `cortex deploy`, or update your cluster by following the instructions at https://docs.cortex.dev/cluster-management/update"
    raise ValueError(errMsg)


API_LIVENESS_UPDATE_PERIOD = 5  # seconds


local_cache = {
    "api": None,
    "provider": None,
    "predictor_impl": None,
    "predict_route": None,
    "client": None,
    "class_set": set(),
}


def update_api_liveness():
    threading.Timer(API_LIVENESS_UPDATE_PERIOD, update_api_liveness).start()
    with open("/mnt/workspace/api_liveness.txt", "w") as f:
        f.write(str(math.ceil(time.time())))


def startup():
    open("/mnt/workspace/api_readiness.txt", "a").close()
    update_api_liveness()


def get_spec(provider, storage, cache_dir, spec_path):
    if provider == "local":
        return read_msgpack(spec_path)

    local_spec_path = os.path.join(cache_dir, "api_spec.msgpack")
    _, key = S3.deconstruct_s3_path(spec_path)
    storage.download_file(key, local_spec_path)
    return read_msgpack(local_spec_path)


def read_msgpack(msgpack_path):
    with open(msgpack_path, "rb") as msgpack_file:
        return msgpack.load(msgpack_file, raw=False)


def sqs_loop():
    sqs = boto3.client("sqs", region_name=os.environ["AWS_REGION"])

    queue_url = os.environ.get("SQS_QUEUE_URL")

    open("/mnt/workspace/api_readiness.txt", "a").close()

    print(queue_url)

    while True:
        print("entering loop")
        response = sqs.receive_message(
            QueueUrl=queue_url,
            AttributeNames=["SentTimestamp"],
            MaxNumberOfMessages=1,
            MessageAttributeNames=["All"],
            WaitTimeSeconds=20,
        )

        print(response)

        if response.get("Messages") is None or len(response["Messages"]) == 0:
            print("no results")
            break

        local_cache["predictor_impl"].predict(payload=response["Messages"][0]["Body"])

        sqs.delete_message(
            QueueUrl=queue_url, ReceiptHandle=response["Messages"][0]["ReceiptHandle"]
        )


def start():
    cache_dir = os.environ["CORTEX_CACHE_DIR"]
    provider = os.environ["CORTEX_PROVIDER"]
    spec_path = os.environ["CORTEX_API_SPEC"]
    project_dir = os.environ["CORTEX_PROJECT_DIR"]
    model_dir = os.getenv("CORTEX_MODEL_DIR", None)
    tf_serving_port = os.getenv("CORTEX_TF_SERVING_PORT", "9000")
    tf_serving_host = os.getenv("CORTEX_TF_SERVING_HOST", "localhost")

    if provider == "local":
        storage = LocalStorage(os.getenv("CORTEX_CACHE_DIR"))
    else:
        storage = S3(bucket=os.environ["CORTEX_BUCKET"], region=os.environ["AWS_REGION"])

    try:
        raw_api_spec = get_spec(provider, storage, cache_dir, spec_path)
        api = API(provider=provider, storage=storage, cache_dir=cache_dir, **raw_api_spec)
        client = api.predictor.initialize_client(
            model_dir, tf_serving_host=tf_serving_host, tf_serving_port=tf_serving_port
        )
        cx_logger().info("loading the predictor from {}".format(api.predictor.path))
        predictor_impl = api.predictor.initialize_impl(project_dir, client)

        local_cache["api"] = api
        local_cache["provider"] = provider
        local_cache["client"] = client
        local_cache["predictor_impl"] = predictor_impl
        local_cache["predict_fn_args"] = inspect.getfullargspec(predictor_impl.predict).args
    except:
        cx_logger().exception("failed to start api")
        sys.exit(1)

    print("starting sqs loop...")
    sqs_loop()


if __name__ == "__main__":
    start()
