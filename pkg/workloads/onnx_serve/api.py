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
import json
import argparse
import traceback
import time
from flask import Flask, request, jsonify
from flask_api import status
from waitress import serve
import onnxruntime as rt
from lib.storage import S3
import numpy

import consts
from lib import util, package, Context
from lib.log import get_logger
from lib.exceptions import CortexException, UserRuntimeException, UserException

logger = get_logger()
logger.propagate = False  # prevent double logging (flask modifies root logger)

app = Flask(__name__)

local_cache = {
    "ctx": None,
    "api": None,
    "sess": None,
    "model_inputs": None,
    "model_outputs": None,
    "request_handler": None,
}


def prediction_failed(sample, reason=None):
    message = "prediction failed for sample: {}".format(json.dumps(sample))
    if reason:
        message += " ({})".format(reason)

    logger.error(message)
    return message, status.HTTP_406_NOT_ACCEPTABLE


@app.route("/healthz", methods=["GET"])
def health():
    return jsonify({"ok": True})


@app.route("/<app_name>/<api_name>", methods=["POST"])
def predict(app_name, api_name):
    try:
        payload = request.get_json()
    except Exception as e:
        return "Malformed JSON", status.HTTP_400_BAD_REQUEST

    sess = local_cache["sess"]
    api = local_cache["api"]
    request_handler = local_cache["request_handler"]
    model_inputs = local_cache["model_inputs"]
    model_outputs = local_cache["model_outputs"]

    response = {}

    if not util.is_dict(payload) or "samples" not in payload:
        util.log_pretty(payload, logging_func=logger.error)
        return prediction_failed(payload, "top level `samples` key not found in request")

    logger.info("Predicting " + util.pluralize(len(payload["samples"]), "sample", "samples"))

    predictions = []
    samples = payload["samples"]
    if not util.is_list(samples):
        util.log_pretty(samples, logging_func=logger.error)
        return prediction_failed(
            payload, "expected the value of key `samples` to be a list of json objects"
        )

    for i, sample in enumerate(payload["samples"]):
        util.log_indent("sample {}".format(i + 1), 2)
        try:
            util.log_indent("Raw sample:", indent=4)
            util.log_pretty(sample, indent=6)
            inference_input = request_handler.preinference(sample, model_inputs)
            labels = [output_node.name for output_node in model_outputs]
            inference = sess.run(labels, inference_input)
            result = request_handler.postinference(inference, model_outputs)
            util.log_indent("Prediction:", indent=4)
            util.log_pretty(result, indent=6)
            prediction = {"prediction": result}
        except CortexException as e:
            e.wrap("error", "sample {}".format(i + 1))
            logger.error(str(e))
            logger.exception("An error occurred, see `cx logs api {}` for more details.".format(1))
            return prediction_failed(sample, str(e))
        except Exception as e:
            logger.exception("An error occurred, see `cx logs api {}` for more details.".format(2))
            return prediction_failed(sample, str(e))

        predictions.append(result)

    response["predictions"] = predictions
    response["resource_id"] = api["id"]

    return jsonify(response)


def start(args):
    logger.info(args)
    ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)
    package.install_packages(ctx.python_packages, ctx.storage)
    api = ctx.apis_id_map[args.api]

    local_cache["api"] = api
    local_cache["ctx"] = ctx
    if api.get("request_handler_impl_key") is not None:
        local_cache["request_handler"], _ = ctx.get_request_handler_impl(api["name"])

    logger.info(ctx)
    model_cache_path = os.path.join(args.model_dir, args.api)
    if not os.path.exists(model_cache_path):
        ctx.storage.download_file_external(api["model"], model_cache_path)

    sess = rt.InferenceSession(model_cache_path)
    local_cache["sess"] = sess
    local_cache["model_inputs"] = sess.get_inputs()
    local_cache["model_outputs"] = sess.get_outputs()
    serve(app, listen="*:{}".format(args.port))
    logger.info("Serving model")


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--workload-id", required=True, help="Workload ID")
    na.add_argument("--port", type=int, required=True, help="Port (on localhost) to use")
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
