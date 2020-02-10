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

import subprocess
import time
import os
import boto3
import os.path
from datetime import datetime
import datadog
import collections
import math

cloudwatch = boto3.client("cloudwatch", region_name=os.environ["AWS_REGION"])
host_ip = os.environ["HOST_IP"]
tick_len = 5


def main():
    # Test cloudwatch access

    # paginator = cloudwatch.get_paginator("list_metrics")
    # for response in paginator.paginate():
    #     print([metric["MetricName"] for metric in response["Metrics"][:3]])  # just get a few

    counter = 0
    statsd = datadog.initialize(statsd_host=host_ip, statsd_port="8125")
    statsd = datadog.statsd
    queue = collections.deque([], tick_len)

    target_start_time = math.ceil(time.time() / 10) * 10 + 2
    time.sleep(target_start_time - time.time())

    while True:
        if not os.path.isfile("/mnt/health_check.txt"):
            print("waiting...", flush=True)
        else:
            break
        time.sleep(10)
    while True:
        cur_time = time.time()
        new_conn = get_open_conn()
        queue.appendleft(new_conn)
        counter = (counter + 1) % tick_len
        if counter == 0:
            open_conn = sum(queue) / len(queue)
            publish_queue(statsd, open_conn)
        sleep_time = cur_time + 2 - time.time()
        if sleep_time > 0:
            time.sleep(sleep_time)


def get_open_conn():
    out = subprocess.run(
        "ss --no-header | grep ':8888 '",
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        shell=True,
        universal_newlines=True,
    )
    # print(out.stdout, flush=True)
    out_stripped = out.stdout.strip()
    if len(out_stripped) == 0:
        open_connections = 0
    else:
        open_connections = len(out_stripped.split("\n"))

    return open_connections


def publish_queue(statsd, open_conn):
    print("NUM CONNECTIONS PUBLISHED: {}".format(open_conn), flush=True)
    # statsd.histogram("in-flight", value=open_conn, tags=["apiName:test"])
    response = cloudwatch.put_metric_data(
        Namespace="cortex",
        MetricData=[
            {
                "MetricName": "in-flight",
                "Dimensions": [{"Name": "apiName", "Value": "test"}],
                "Timestamp": datetime.now(),
                "Value": open_conn,
                "Unit": "Count",
                "StorageResolution": 1,
            }
        ],
    )


if __name__ == "__main__":
    main()
