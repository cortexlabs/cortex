"""
Typical usage example:

    python submit.py <cortex-env> <dest-s3-dir>
"""

from typing import List

import sys
import json
import requests
import cortex


def main():
    # parse args
    if len(sys.argv) != 3:
        print("usage: python submit.py <cortex-env> <dest-s3-dir>")
        sys.exit(1)
    env_name = sys.argv[1]
    dest_s3_dir = sys.argv[2]

    # read sample file
    with open("sample.json") as f:
        sample_items: List[str] = json.load(f)

    # get batch endpoint
    cx = cortex.client(env_name)
    batch_endpoint = cx.get_api("image-classifier-alexnet")["endpoint"]

    # submit job
    job_spec = {
        "workers": 1,
        "item_list": {"items": sample_items, "batch_size": 1},
        "config": {"dest_s3_dir": dest_s3_dir},
    }
    response = requests.post(batch_endpoint, json=job_spec)
    print(json.dumps(response.json(), indent=2))


if __name__ == "__main__":
    main()
