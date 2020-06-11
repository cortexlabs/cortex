# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import os
import click
import concurrent.futures
import requests
import signal
import json
import time
import itertools
import cv2
import numpy as np
import base64


@click.command(
    help=(
        "Program for testing the throughput of Resnet50 model on "
        "instances equipped with CPU, GPU or Inferentia ASIC devices."
    )
)
@click.argument("img_url", type=str, envvar="IMG_URL")
@click.argument("endpoint", type=str, envvar="ENDPOINT")
@click.option(
    "--workers",
    "-w",
    type=int,
    default=1,
    show_default=True,
    help="Number of workers for prediction requests.",
)
@click.option(
    "--threads", "-t", type=int, default=1, show_default=True, help="Number of threads per worker."
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
    help="How long the thread making predictions will run for in seconds. If set, -s option won't be considered anymore.",
)
@click.option(
    "--batch-size",
    "-b",
    type=int,
    default=1,
    show_default=True,
    help="Number of images sent for inference in one request.",
)
def main(img_url, endpoint, workers, threads, samples, time_based, batch_size):
    # get the image in bytes representation
    image = get_url_image(img_url)
    image_bytes = image_to_jpeg_bytes(image)

    # encode image
    image_enc = base64.b64encode(image_bytes).decode("utf-8")
    images_enc = [image_enc for i in range(batch_size)]
    data = json.dumps({"imgs": images_enc})

    print("Starting the inference throughput test...")
    results = []
    start = time.time()
    with concurrent.futures.ProcessPoolExecutor(max_workers=workers) as executor:
        results = executor_submitter(
            executor, workers, process_worker, threads, data, endpoint, samples, time_based
        )
    end = time.time()
    elapsed = end - start

    total_requests = sum(results) * batch_size

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

    if time_based == 0.0:
        for i in range(samples):
            try:
                resp = requests.post(
                    endpoint,
                    data=data,
                    headers={"content-type": "application/json"},
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
                    headers={"content-type": "application/json"},
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


def get_url_image(url_image):
    """
    Get numpy image from URL image.
    """
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_COLOR)
    return image


if __name__ == "__main__":
    main()
