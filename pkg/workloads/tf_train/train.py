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
import traceback
import tensorflow as tf

from lib import util, package, Context
from lib.exceptions import CortexException, UserRuntimeException
import train_util

from lib.log import get_logger

logger = get_logger()


def train(args):
    ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)

    package.install_packages(ctx.python_packages, ctx.storage)

    model = ctx.models_id_map[args.model]

    logger.info("Training")

    with util.Tempdir(ctx.cache_dir) as temp_dir:
        model_dir = os.path.join(temp_dir, "model_dir")
        ctx.upload_resource_status_start(model)

        try:
            estimator_impl, _ = ctx.get_estimator_impl(model["name"])
            train_util.train(model["name"], estimator_impl, ctx, model_dir)
            ctx.upload_resource_status_success(model)

            logger.info("Caching")
            logger.info("Caching model " + model["name"])
            model_export_dir = os.path.join(model_dir, "export", "estimator")
            model_zip_path = os.path.join(temp_dir, "model.zip")
            util.zip_dir(model_export_dir, model_zip_path)

            ctx.storage.upload_file(model_zip_path, model["key"])
            util.log_job_finished(ctx.workload_id)

        except CortexException as e:
            ctx.upload_resource_status_failed(model)
            e.wrap("error")
            logger.error(str(e))
            logger.exception(
                "An error occurred, see `cx logs model {}` for more details.".format(model["name"])
            )
            sys.exit(1)
        except Exception as e:
            ctx.upload_resource_status_failed(model)
            logger.exception(
                "An error occurred, see `cx logs model {}` for more details.".format(model["name"])
            )
            sys.exit(1)


def main():
    logger.info("Starting")

    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--workload-id", required=True, help="Workload ID")
    na.add_argument(
        "--context", required=True, help="S3 path to context (e.g. s3://bucket/path/to/context.json"
    )
    na.add_argument("--cache-dir", required=True, help="Local path for the context cache")
    na.add_argument("--model", required=True, help="Resource id of the model to train")
    parser.set_defaults(func=train)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
