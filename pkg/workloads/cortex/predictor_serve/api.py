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
import argparse
import time

from flask import Flask, request, jsonify, g
from flask_api import status
from waitress import serve

from cortex.lib import util, Context, api_utils
from cortex.lib.log import get_logger, debug_obj
from cortex.lib.exceptions import CortexException, UserRuntimeException

logger = get_logger()
logger.propagate = False  # prevent double logging (flask modifies root logger)

app = Flask(__name__)

app.json_encoder = util.json_tricks_encoder

local_cache = {"ctx": None, "api": None, "class_set": set()}


@app.before_request
def before_request():
    g.start_time = time.time()


@app.after_request
def after_request(response):
    response.headers["Access-Control-Allow-Origin"] = "*"
    response.headers["Access-Control-Allow-Headers"] = "*"

    if request.path != "/predict":
        return response

    api = local_cache["api"]
    ctx = local_cache["ctx"]

    logger.info(response.status)

    prediction = None
    if "prediction" in g:
        prediction = g.prediction

    api_utils.post_request_metrics(
        ctx, api, response, prediction, g.start_time, local_cache["class_set"]
    )

    return response


def prediction_failed(reason):
    message = "prediction failed: {}".format(reason)
    logger.error(message)
    return message, status.HTTP_406_NOT_ACCEPTABLE


@app.route("/healthz", methods=["GET"])
def health():
    return jsonify({"ok": True})


@app.route("/predict", methods=["POST"])
def predict():
    debug = request.args.get("debug", "false").lower() == "true"

    try:
        sample = request.get_json()
    except:
        return "malformed json", status.HTTP_400_BAD_REQUEST

    api = local_cache["api"]
    predictor = local_cache["predictor"]

    try:
        try:
            debug_obj("sample", sample, debug)
            output = predictor.predict(sample, api["predictor"]["metadata"])
            debug_obj("prediction", output, debug)
        except Exception as e:
            raise UserRuntimeException(api["predictor"]["path"], "predict", str(e)) from e
    except Exception as e:
        logger.exception("prediction failed")
        return prediction_failed(str(e))

    g.prediction = output
    return jsonify(output)


@app.errorhandler(Exception)
def exceptions(e):
    logger.exception(e)
    return jsonify(error=str(e)), 500


def start(args):
    api = None
    try:
        ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)
        api = ctx.apis_id_map[args.api]
        local_cache["api"] = api
        local_cache["ctx"] = ctx

        if api.get("predictor") is None:
            raise CortexException(api["name"], "predictor key not configured")

        local_cache["predictor"] = ctx.get_predictor_impl(api["name"], args.project_dir)

        if util.has_function(local_cache["predictor"], "init"):
            try:
                local_cache["predictor"].init(api["predictor"]["metadata"])
            except Exception as e:
                raise UserRuntimeException(api["predictor"]["path"], "init", str(e)) from e
        logger.info("init ran successfully")
    except:
        logger.exception("failed to start api")
        sys.exit(1)

    if api.get("tracker") is not None and api["tracker"].get("model_type") == "classification":
        try:
            local_cache["class_set"] = api_utils.get_classes(ctx, api["name"])
        except Exception as e:
            logger.warn("an error occurred while attempting to load classes", exc_info=True)

    serve(app, listen="*:{}".format(args.port))


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
    na.add_argument("--cache-dir", required=True, help="Local path for the context cache")
    na.add_argument("--project-dir", required=True, help="Local path for the project zip file")

    parser.set_defaults(func=start)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
