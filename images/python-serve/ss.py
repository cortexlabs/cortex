import subprocess
import time
import os
import boto3


cloudwatch = boto3.client("cloudwatch", region_name=os.environ["AWS_REGION"])


def main():
    # Test cloudwatch access
    paginator = cloudwatch.get_paginator("list_metrics")
    for response in paginator.paginate():
        print([metric["MetricName"] for metric in response["Metrics"][:3]])  # just get a few

    while True:
        out = subprocess.run(
            "ss --no-header | grep ':8888 ' | wc -l",
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            shell=True,
            universal_newlines=True,
        )

        print("NUM CONNECTIONS: {}".format(out.stdout.strip()), flush=True)

        time.sleep(2)


if __name__ == "__main__":
    main()
