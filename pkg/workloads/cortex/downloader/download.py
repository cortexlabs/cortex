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
from cortex.lib.log import get_logger

logger = get_logger()


def start(args):
    download = json.loads(base64.urlsafe_b64decode(args.download))
    for download_arg in download:
        from_path = download_arg["from"]
        to_path = download_arg["to"]
        bucket_name, prefix = S3.deconstruct_s3_path(from_path)
        s3_client = S3(bucket_name, client_config={})
        s3_client.download(prefix, to_path)
        if download_arg["unzip"] and os.path.basename(from_path).endswith("zip"):
            util.extract_zip(
                os.path.join(to_path, os.path.basename(from_path)), delete_zip_file=True
            )


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--download", required=True, help="path_to_download_from;path_to_download_to")
    parser.set_defaults(func=start)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
