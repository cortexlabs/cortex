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

import os
import sys
import argparse
from subprocess import run

from lib import util, aws
from lib.context import Context
from lib.log import get_logger
from lib.exceptions import UserException, CortexException

logger = get_logger()

LOCAL_PACKAGE_PATH = "/src/package"
WHEELHOUSE_PATH = "/wheelhouse"


def get_build_order(python_packages):
    build_order = []
    if "requirements.txt" in python_packages:
        build_order.append("requirements.txt")
    return build_order + sorted([name for name in python_packages if name != "requirements.txt"])


def build_packages(python_packages, bucket):
    requirements_map = {}
    build_order = get_build_order(python_packages)
    for package_name in build_order:
        python_package = python_packages[package_name]
        if package_name == "requirements.txt":
            requirements_path = os.path.join(LOCAL_PACKAGE_PATH, python_package["name"])
            aws.download_file_from_s3(python_package["raw_key"], requirements_path, bucket)
            requirements_map[python_package["name"]] = "-r " + requirements_path
        else:
            aws.download_and_extract_zip(python_package["raw_key"], LOCAL_PACKAGE_PATH, bucket)
            requirements_map[python_package["name"]] = os.path.join(
                LOCAL_PACKAGE_PATH, python_package["name"]
            )

    logger.info("Setting up packages")

    for package_name in build_order:
        requirement = requirements_map[package_name]
        logger.info("Building package {}".format(package_name))
        completed_process = run(
            "pip3 wheel -w {} {}".format(
                os.path.join(WHEELHOUSE_PATH, package_name), requirement
            ).split()
        )
        if completed_process.returncode != 0:
            raise UserException("creating wheels", package_name)

    logger.info("Validating packages")

    for package_name in build_order:
        requirement = requirements_map[package_name]
        logger.info("Installing package {}".format(package_name))
        completed_process = run(
            "pip3 install --no-index --find-links={} {}".format(
                os.path.join(WHEELHOUSE_PATH, package_name), requirement
            ).split()
        )
        if completed_process.returncode != 0:
            raise UserException("installing package", package_name)

    logger.info("Caching built packages")

    for package_name in build_order:
        aws.compress_zip_and_upload(
            os.path.join(WHEELHOUSE_PATH, package_name),
            python_packages[package_name]["package_key"],
            bucket,
        )


def build(args):
    ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)
    python_packages_list = [ctx.pp_id_map[id] for id in args.python_packages.split(",")]
    python_packages = {
        python_package["name"]: python_package for python_package in python_packages_list
    }
    ctx.upload_resource_status_start(*python_packages_list)
    try:
        build_packages(python_packages, ctx.bucket)
    except Exception as e:
        logger.exception(e)
        ctx.upload_resource_status_failed(*python_packages_list)
    else:
        ctx.upload_resource_status_success(*python_packages_list)


def install_packages(python_packages, bucket):
    build_order = get_build_order(python_packages)

    for package_name in build_order:
        python_package = python_packages[package_name]
        aws.download_and_extract_zip(
            python_package["package_key"], os.path.join(WHEELHOUSE_PATH, package_name), bucket
        )

    aws.download_file_from_s3(
        python_packages["requirements.txt"]["raw_key"], "/requirements.txt", bucket
    )

    for package_name in build_order:
        cmd = package_name
        if package_name == "requirements.txt":
            cmd = "-r /requirements.txt"

        completed_process = run(
            "pip3 install --no-cache-dir --no-index --find-links={} {}".format(
                os.path.join(WHEELHOUSE_PATH, package_name), cmd
            ).split()
        )
        if completed_process.returncode != 0:
            raise UserException("installing package", package_name)

    util.rm_file("/requirements.txt")
    util.rm_dir(WHEELHOUSE_PATH)


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--workload-id", required=True, help="Workload ID")
    na.add_argument(
        "--context", required=True, help="S3 path to context (e.g. s3://bucket/path/to/context.json"
    )
    na.add_argument("--cache-dir", required=True, help="Local path for the context cache")
    na.add_argument("--python-packages", help="Resource ids of packages to build")
    na.add_argument("--build", action="store_true", help="sum the integers (default: find the max)")

    args, _ = parser.parse_known_args()
    if args.build:
        build(args)
    else:
        ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)
        install_packages(ctx.python_packages, ctx.bucket)


if __name__ == "__main__":
    main()
