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

from flask import Flask, request, jsonify, g
from flask_api import status
from waitress import serve

from cortex.lib import util, Context, api_utils
from cortex.lib.log import cx_logger, debug_obj
from cortex.lib.exceptions import CortexException, UserRuntimeException, UserException
from cortex.onnx_serve.client import ONNXClient

app = Flask(__name__)

app.json_encoder = util.json_tricks_encoder

local_cache = {"ctx": None, "api": None, "client": None, "class_set": set()}


@app.before_request
def before_request():
    g.start_time = time.time()


@app.after_request
def after_request(response):
    response.headers["Access-Control-Allow-Origin"] = "*"
    response.headers["Access-Control-Allow-Headers"] = request.headers.get(
        "Access-Control-Request-Headers", "*"
    )

    if not (request.path == "/predict" and request.method == "POST"):
        return response

    api = local_cache["api"]
    ctx = local_cache["ctx"]

    cx_logger().info(response.status)

    prediction = None
    if "prediction" in g:
        prediction = g.prediction

    api_utils.post_request_metrics(
        ctx, api, response, prediction, g.start_time, local_cache["class_set"]
    )

    return response


@app.route("/predict", methods=["POST"])
def predict():
    debug = request.args.get("debug", "false").lower() == "true"

    try:
        payload = request.get_json()
    except:
        return "malformed json", status.HTTP_400_BAD_REQUEST

    api = local_cache["api"]
    predictor = local_cache["predictor"]

    try:
        debug_obj("payload", payload, debug)
        try:
            output = predictor.predict(payload)
        except Exception as e:
            raise UserRuntimeException(api["onnx"]["predictor"], "predict", str(e)) from e
        debug_obj("prediction", output, debug)

    except Exception as e:
        cx_logger().exception("prediction failed")
        return prediction_failed(str(e))

    g.prediction = output
    return jsonify(output)


def prediction_failed(reason):
    message = "prediction failed: {}".format(reason)
    cx_logger().error(message)
    return message, status.HTTP_406_NOT_ACCEPTABLE


@app.route("/predict", methods=["GET"])
def get_summary():
    response = {
        "model_signature": local_cache["client"].input_signature,
        "message": api_utils.API_SUMMARY_MESSAGE,
    }
    return jsonify(response)


@app.errorhandler(Exception)
def exceptions(e):
    cx_logger().exception(e)
    return jsonify(error=str(e)), 500


def start(args):
    api = None
    try:
        ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)
        api = ctx.apis_id_map[args.api]
        local_cache["api"] = api
        local_cache["ctx"] = ctx

        if api.get("onnx") is None:
            raise CortexException(api["name"], "onnx key not configured")

        cx_logger().info("loading the predictor from {}".format(api["onnx"]["predictor"]))

        _, prefix = ctx.storage.deconstruct_s3_path(api["onnx"]["model"])
        model_path = os.path.join(args.model_dir, os.path.basename(prefix))
        local_cache["client"] = ONNXClient(model_path)

        predictor_class = ctx.get_predictor_class(api["name"], args.project_dir)

        local_cache["predictor"] = predictor_class(local_cache["client"], api["onnx"]["config"])
    except Exception as e:
        cx_logger().exception("failed to start api")
        sys.exit(1)

    if api.get("tracker") is not None and api["tracker"].get("model_type") == "classification":
        try:
            local_cache["class_set"] = api_utils.get_classes(ctx, api["name"])
        except Exception as e:
            cx_logger().warn("an error occurred while attempting to load classes", exc_info=True)

    cx_logger().info("ONNX model signature: {}".format(local_cache["client"].input_signature))

    waitress_kwargs = {}
    if api["onnx"].get("config") is not None:
        for key, value in api["onnx"]["config"].items():
            if key.startswith("waitress_"):
                waitress_kwargs[key[len("waitress_") :]] = value

    if len(waitress_kwargs) > 0:
        cx_logger().info("waitress parameters: {}".format(waitress_kwargs))

    waitress_kwargs["listen"] = "*:{}".format(args.port)

    cx_logger().info("{} api is live".format(api["name"]))
    open("/health_check.txt", "a").close()
    serve(app, **waitress_kwargs)


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--workload-id", required=True, help="workload id")
    na.add_argument("--port", type=int, required=True, help="port (on localhost) to use")
    na.add_argument(
        "--context",
        required=True,
        help="s3 path to context (e.g. s3://bucket/path/to/context.json)",
    )
    na.add_argument("--api", required=True, help="resource id of api to serve")
    na.add_argument("--model-dir", required=True, help="directory to download the model to")
    na.add_argument("--cache-dir", required=True, help="local path for the context cache")
    na.add_argument("--project-dir", required=True, help="local path for the project zip file")
    parser.set_defaults(func=start)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
