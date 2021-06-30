"""
Typical usage example:

    python submit.py <cortex-env> <workers>
"""

from typing import List

import sys
import json
import requests
import cortex


def main():
    # parse args
    if len(sys.argv) != 3:
        print("usage: python submit.py <cortex-env> <workers>")
        sys.exit(1)
    env_name = sys.argv[1]
    workers = int(sys.argv[2])

    # get batch endpoint
    cx = cortex.client(env_name)
    batch_endpoint = cx.get_api("golang")["endpoint"]

    # submit job
    job_spec = {"workers": workers, "item_list": {"items": [1] * workers, "batch_size": 1}}
    response = requests.post(batch_endpoint, json=job_spec)
    print(json.dumps(response.json(), indent=2))


if __name__ == "__main__":
    main()
