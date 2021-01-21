# Copyright 2021 Cortex Labs, Inc.
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

from cortex_internal.lib import util
from cortex_internal.lib.storage import S3, GCS
from cortex_internal.lib.log import configure_logger

util.expand_environment_vars_on_file(os.environ["CORTEX_LOG_CONFIG_FILE"])
logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


def start(args):
    download_config = json.loads(base64.urlsafe_b64decode(args.download))
    for download_arg in download_config["download_args"]:
        from_path = download_arg["from"]
        to_path = download_arg["to"]
        item_name = download_arg.get("item_name", "")

        if from_path.startswith("s3://"):
            bucket_name, prefix = S3.deconstruct_s3_path(from_path)
            client = S3(bucket_name)
        elif from_path.startswith("gs://"):
            bucket_name, prefix = GCS.deconstruct_gcs_path(from_path)
            client = GCS(bucket_name)
        else:
            raise ValueError('"from" download arg can either have the "s3://" or "gs://" prefixes')

        if item_name != "":
            if download_arg.get("hide_from_log", False):
                logger.info("downloading {}".format(item_name))
            else:
                logger.info("downloading {} from {}".format(item_name, from_path))

        if download_arg.get("to_file", False):
            client.download_file(prefix, to_path)
        else:
            client.download(prefix, to_path)

        if download_arg.get("unzip", False):
            if item_name != "" and not download_arg.get("hide_unzipping_log", False):
                logger.info("unzipping {}".format(item_name))
            if download_arg.get("to_file", False):
                util.extract_zip(to_path, delete_zip_file=True)
            else:
                util.extract_zip(
                    os.path.join(to_path, os.path.basename(from_path)), delete_zip_file=True
                )

    if download_config.get("last_log", "") != "":
        logger.info(download_config["last_log"])


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument(
        "--download",
        required=True,
        help="a base64 encoded download_config (see k8s_specs.go for the structure)",
    )
    parser.set_defaults(func=start)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
