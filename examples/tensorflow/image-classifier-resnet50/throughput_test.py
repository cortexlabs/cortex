import os
import click
import concurrent.futures
import requests
import signal
import json
import time
import itertools


@click.command(
    help=(
        "Program for testing the throughput of Resnet50 model on "
        "instances equipped with CPU, GPU or Accelerator devices."
    )
)
@click.argument("img_url_src", type=str, envvar="IMG_URL")
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
    help="How long the thread makes prediction in seconds. If set, -s option won't be considered anymore.",
)
def main(img_url_src, endpoint, workers, threads, samples, time_based):
    results = []
    start = time.time()
    with concurrent.futures.ProcessPoolExecutor(max_workers=workers) as executor:
        results = executor_submitter(
            executor, workers, process_worker, threads, img_url_src, endpoint, samples, time_based
        )
    end = time.time()
    elapsed = end - start

    total_requests = sum(results)
    if time_based > 0.0:
        print(f"A total of {total_requests} requests have been served in {time_based} seconds")
        print(f"Avg number of inferences/sec is {total_requests / time_based}")
        print(f"Avg time spent on an inference is {time_based / total_requests} seconds")
    else:
        print(f"A total of {total_requests} requests have been served in {elapsed} seconds")
        print(f"Avg number of inferences/sec is {total_requests / elapsed}")
        print(f"Avg time spent on an inference is {elapsed / total_requests} seconds")


def process_worker(threads, img_url_src, endpoint, samples, time_based):
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=threads) as executor:
        results = executor_submitter(
            executor, threads, task, img_url_src, endpoint, samples, time_based
        )

    return results


def executor_submitter(executor, workers, *args, **kwargs):
    futures = []
    for worker in range(workers):
        future = executor.submit(*args, **kwargs)
        futures.append(future)

    results = [future.result() for future in futures]
    results = list(itertools.chain.from_iterable(results))

    return results


def task(img_url_src, endpoint, samples, time_based):
    data = json.dumps({"url": img_url_src,})
    timeout = 15

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
            time.sleep(0.005)
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
            time.sleep(0.005)
            counter += 1
        return [counter]


if __name__ == "__main__":
    main()
