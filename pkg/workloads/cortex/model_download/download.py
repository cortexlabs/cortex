# Copyright 2019 Cortex Labs, Inc.
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
import time

from cortex.lib import Context, api_utils
from cortex.lib.storage import S3
from cortex.lib.log import get_logger
from cortex.lib.exceptions import UserRuntimeException, UserException

logger = get_logger()
logger.propagate = False  # prevent double logging (flask modifies root logger)


def start(args):
    api = None
    try:
        ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)
        api = ctx.apis_id_map[args.api]

        bucket_name, prefix = ctx.storage.deconstruct_s3_path(api["model"])
        s3_client = S3(bucket_name, client_config={})
        s3_client.download_dir(prefix, args.model_dir)

    except Exception as e:
        logger.exception(
            "An error occurred, see `cortex logs -v api {}` for more details.".format(api["name"])
        )
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--workload-id", required=True, help="Workload ID")
    na.add_argument(
        "--context",
        required=True,
        help="S3 path to context (e.g. s3://bucket/path/to/context.json)",
    )
    na.add_argument("--api", required=True, help="Resource id of api to serve")
    na.add_argument("--model-dir", required=True, help="Directory to download the model to")
    na.add_argument("--cache-dir", required=True, help="Local path for the context cache")
    parser.set_defaults(func=start)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
