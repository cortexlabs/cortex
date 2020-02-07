import subprocess
import time
import os
import boto3
import os.path
from datetime import datetime
import datadog
import collections

cloudwatch = boto3.client("cloudwatch", region_name=os.environ["AWS_REGION"])
host_ip = os.environ["HOST_IP"]
print(host_ip)
tick_len = 10


def main():
    # Test cloudwatch access

    # paginator = cloudwatch.get_paginator("list_metrics")
    # for response in paginator.paginate():
    #     print([metric["MetricName"] for metric in response["Metrics"][:3]])  # just get a few

    while True:
        if not os.path.isfile("/mnt/health_check.txt"):
            print("waiting...", flush=True)
        else:
            break
        time.sleep(1)

    counter = 0
    statsd = datadog.initialize(statsd_host=host_ip, statsd_port="8125")
    statsd = datadog.statsd
    queue = collections.deque([], tick_len)
    while True:
        cur_time = time.time()
        new_conn = get_open_conn()
        queue.appendleft(new_conn)
        counter = (counter + 1) % tick_len
        if counter == 0:
            open_conn = sum(queue) / len(queue)
            publish_queue(statsd, open_conn)
        sleep_time = cur_time + 1.0 - time.time()
        time.sleep(1)


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
