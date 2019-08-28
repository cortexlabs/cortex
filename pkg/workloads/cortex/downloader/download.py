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

import argparse

from cortex.lib.storage import S3
from cortex.lib.log import get_logger

logger = get_logger()


def start(args):
    bucket_name, prefix = S3.deconstruct_s3_path(args.download_from)
    s3_client = S3(bucket_name, client_config={})
    logger.info(args.download_to)
    s3_client.download(prefix, args.download_to)


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--download_from", required=True, help="Storage Path to download the file from")
    na.add_argument("--download_to", required=True, help="Directory to download the file to")
    parser.set_defaults(func=start)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
