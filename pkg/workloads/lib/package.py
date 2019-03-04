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
import glob
from subprocess import run

from lib import util, aws
from lib.context import Context
from lib.log import get_logger
from lib.exceptions import UserException, CortexException

import requirements

logger = get_logger()

LOCAL_PACKAGE_PATH = "/src/package"
WHEELHOUSE_PATH = "/wheelhouse"


def get_build_order(python_packages):
    build_order = []
    if "requirements.txt" in python_packages:
        build_order.append("requirements.txt")
    return build_order + sorted([name for name in python_packages if name != "requirements.txt"])


def get_restricted_packages():
    cortex_packages = {"pyspark": "2.4.0", "tensorflow": "1.12.0"}
    req_files = glob.glob("/src/**/requirements.txt", recursive=True)
    for req_file in req_files:
        with open(req_file) as f:
            for req in requirements.parse(f):
                cortex_packages[req.name] = req.specs[0][1]
    return cortex_packages


def build_packages(python_packages, bucket):
    cmd_partial = {}
    build_order = get_build_order(python_packages)
    for package_name in build_order:
        python_package = python_packages[package_name]
        if package_name == "requirements.txt":
            requirements_path = os.path.join(LOCAL_PACKAGE_PATH, package_name)
            aws.download_file_from_s3(python_package["src_key"], requirements_path, bucket)
            cmd_partial[package_name] = "-r " + requirements_path
        else:
            aws.download_and_extract_zip(python_package["src_key"], LOCAL_PACKAGE_PATH, bucket)
            cmd_partial[package_name] = os.path.join(LOCAL_PACKAGE_PATH, package_name)

    logger.info("Setting up packages")

    restricted_packages = get_restricted_packages()

    for package_name in build_order:
        package_wheel_path = os.path.join(WHEELHOUSE_PATH, package_name)
        requirement = cmd_partial[package_name]
        logger.info("Building package {}".format(package_name))
        completed_process = run(
            "pip3 wheel -w {} {}".format(package_wheel_path, requirement).split()
        )
        if completed_process.returncode != 0:
            raise UserException("creating wheels", package_name)

        for wheelname in os.listdir(package_wheel_path):
            name_split = wheelname.split("-")
            dist_name, version = name_split[0], name_split[1]
            expected_version = restricted_packages.get(dist_name, None)
            if expected_version is not None and version != expected_version:
                raise UserException(
                    "when installing {}, found {}=={} but cortex requires {}=={}".format(
                        package_name, dist_name, version, dist_name, expected_version
                    )
                )

    logger.info("Validating packages")

    for package_name in build_order:
        requirement = cmd_partial[package_name]
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
        util.log_job_finished(ctx.workload_id)
    except CortexException as e:
        e.wrap("error")
        logger.exception(e)
        ctx.upload_resource_status_failed(*python_packages_list)
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

    if "requirements.txt" in python_packages:
        aws.download_file_from_s3(
            python_packages["requirements.txt"]["src_key"], "/requirements.txt", bucket
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
    na.add_argument(
        "--build", action="store_true", help="Flag to determine mode (build vs install)"
    )

    args, _ = parser.parse_known_args()
    if args.build:
        build(args)
    else:
        ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)
        install_packages(ctx.python_packages, ctx.bucket)


if __name__ == "__main__":
    main()
