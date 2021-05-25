"""
Typical usage example:

    python submit.py <cortex-env> <dest-s3-dir>
"""

import sys
import json
import requests
import cortex

def main():
    # parse args
    if len(sys.argv) < 2:
        print("cortex environment name <env-name> arg required")
        sys.exit(1)
    if len(sys.argv) < 3:
        print("destination bucket <dest-s3-dir> arg required")
        sys.exit(1)
    env_name = sys.argv[1]
    dest_s3_dir = sys.argv[2]

    # get task endpoint
    cx = cortex.client(env_name)
    task_endpoint = cx.get_api("trainer")["endpoint"]

    # submit job
    job_spec = {
        "config": {
            "dest_s3_dir": dest_s3_dir
        }
    }
    response = requests.post(task_endpoint, json=job_spec)
    print(json.dumps(response.json(), indent=2))

if __name__ == '__main__':
    main()
