import os
import sys
import click
import concurrent.futures
import requests
import imageio
import json
import time
import itertools
import cv2
import numpy as np

from validator_collection import checkers


@click.command(help="Program for testing the throughput of Cortex-deployed APIs.")
@click.argument("endpoint", type=str, envvar="ENDPOINT")
@click.argument("payload", type=str, envvar="PAYLOAD")
@click.option(
    "--processes",
    "-p",
    type=int,
    default=1,
    show_default=True,
    help="Number of processes for prediction requests.",
)
@click.option(
    "--threads",
    "-t",
    type=int,
    default=1,
    show_default=True,
    help="Number of threads per process for prediction requests.",
)
@click.option(
    "--samples",
    "-s",
    type=int,
    default=10,
    show_default=True,
    help="Number of samples to run per thread.",
)
@click.option(
    "--time-based",
    "-i",
    type=float,
    default=0.0,
    help="How long the thread making predictions will run for in seconds. If set, -s option will be ignored.",
)
def main(payload, endpoint, processes, threads, samples, time_based):
    file_type = None
    if checkers.is_url(payload):
        if payload.lower().endswith(".json"):
            file_type = "json"
            payload_data = requests.get(payload).json()
        elif payload.lower().endswith(".jpg"):
            file_type = "jpg"
            payload_data = imageio.imread(payload)
    elif checkers.is_file(payload):
        if payload.lower().endswith(".json"):
            file_type = "json"
            with open(payload, "r") as f:
                payload_data = json.load(f)
        elif payload.lower().endswith(".jpg"):
            file_type = "jpg"
            payload_data = cv2.imread(payload, cv2.IMREAD_COLOR)
    else:
        print(f"'{payload}' isn't an URL resource, nor is it a local file")
        sys.exit(1)

    if file_type is None:
        print(f"'{payload}' doesn't point to a jpg image or to a json file")
        sys.exit(1)
    if file_type == "jpg":
        data = image_to_jpeg_bytes(payload_data)
    if file_type == "json":
        data = json.dumps(payload_data)

    print("Starting the inference throughput test...")
    results = []
    start = time.time()
    with concurrent.futures.ProcessPoolExecutor(max_workers=processes) as executor:
        results = executor_submitter(
            executor, processes, process_worker, threads, data, endpoint, samples, time_based
        )
    end = time.time()
    elapsed = end - start

    total_requests = sum(results)

    print(f"A total of {total_requests} requests have been served in {elapsed} seconds")
    print(f"Avg number of inferences/sec is {total_requests / elapsed}")
    print(f"Avg time spent on an inference is {elapsed / total_requests} seconds")


def process_worker(threads, data, endpoint, samples, time_based):
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=threads) as executor:
        results = executor_submitter(executor, threads, task, data, endpoint, samples, time_based)

    return results


def executor_submitter(executor, workers, *args, **kwargs):
    futures = []
    for worker in range(workers):
        future = executor.submit(*args, **kwargs)
        futures.append(future)

    results = [future.result() for future in futures]
    results = list(itertools.chain.from_iterable(results))

    return results


def task(data, endpoint, samples, time_based):
    timeout = 60

    if isinstance(data, str):
        headers = {"content-type": "application/json"}
    elif isinstance(data, bytes):
        headers = {"content-type": "application/octet-stream"}
    else:
        return

    if time_based == 0.0:
        for i in range(samples):
            try:
                resp = requests.post(
                    endpoint,
                    data=data,
                    headers=headers,
                    timeout=timeout,
                )
            except Exception as e:
                print(e)
                break
            time.sleep(0.1)
        return [samples]
    else:
        start = time.time()
        counter = 0
        while start + time_based >= time.time():
            try:
                resp = requests.post(
                    endpoint,
                    data=data,
                    headers=headers,
                    timeout=timeout,
                )
            except Exception as e:
                print(e)
                break
            time.sleep(0.1)
            counter += 1
        return [counter]


def image_to_jpeg_nparray(image, quality=[int(cv2.IMWRITE_JPEG_QUALITY), 95]):
    """
    Convert numpy image to jpeg numpy vector.
    """
    is_success, im_buf_arr = cv2.imencode(".jpg", image, quality)
    return im_buf_arr


def image_to_jpeg_bytes(image, quality=[int(cv2.IMWRITE_JPEG_QUALITY), 95]):
    """
    Convert numpy image to bytes-encoded jpeg image.
    """
    buf = image_to_jpeg_nparray(image, quality)
    byte_im = buf.tobytes()
    return byte_im


if __name__ == "__main__":
    main()
