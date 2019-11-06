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
import os
import base64
import json

from cortex.lib import util
from cortex.lib.storage import S3
from cortex.lib.log import cx_logger


def start(args):
    download = json.loads(base64.urlsafe_b64decode(args.download))
    for download_arg in download:
        from_path = download_arg["from"]
        to_path = download_arg["to"]
        item_name = download_arg.get("item_name", "")
        bucket_name, prefix = S3.deconstruct_s3_path(from_path)
        s3_client = S3(bucket_name, client_config={})

        if item_name != "":
            cx_logger().info("downloading {} from {}".format(item_name, from_path))
        s3_client.download(prefix, to_path)

        if download_arg["unzip"]:
            if item_name != "":
                cx_logger().info("unzipping {}".format(item_name))
            util.extract_zip(
                os.path.join(to_path, os.path.basename(from_path)), delete_zip_file=True
            )


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument(
        "--download",
        required=True,
        help="a base64 encoded json array of download arg objects (see api_workload.go for the structure)",
    )
    parser.set_defaults(func=start)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
