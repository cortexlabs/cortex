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

from cortex import consts
from cortex.lib import util
from cortex.lib.type import API
from cortex.lib.log import cx_logger, debug_obj
from cortex.lib.storage import S3
from cortex.lib.exceptions import UserRuntimeException

app = Flask(__name__)
app.json_encoder = util.json_tricks_encoder

API_SUMMARY_MESSAGE = (
    "make a prediction by sending a post request to this endpoint with a json payload"
)

local_cache = {"api": None, "predictor_impl": None, "client": None, "class_set": set()}


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

    cx_logger().info(response.status)

    prediction = None
    if "prediction" in g:
        prediction = g.prediction

    try:
        predicted_value = None
        if api.tracker is not None:
            predicted_value = api.tracker.extract_predicted_value(prediction)
            api.post_request_metrics(response.status_code, g.start_time, predicted_value)
            if predicted_value is not None and predicted_value not in local_cache["class_set"]:
                api.upload_class(predicted_value)
                local_cache["class_set"].add(predicted_value)
    except Exception as e:
        cx_logger().warn("unable to record prediction metric", exc_info=True)

    return response


@app.route("/predict", methods=["POST"])
def predict():
    debug = request.args.get("debug", "false").lower() == "true"

    try:
        payload = request.get_json()
    except:
        return "malformed json", status.HTTP_400_BAD_REQUEST

    api = local_cache["api"]
    predictor_impl = local_cache["predictor_impl"]

    try:
        debug_obj("payload", payload, debug)
        try:
            output = predictor_impl.predict(payload)
        except Exception as e:
            raise UserRuntimeException(api.predictor.path, "predict", str(e)) from e
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
    response = {"message": API_SUMMARY_MESSAGE}

    if hasattr(local_cache["client"], "input_signature"):
        response["model_signature"] = local_cache["client"].input_signature
    return jsonify(response)


@app.errorhandler(Exception)
def exceptions(e):
    cx_logger().exception(e)
    return jsonify(error=str(e)), 500


def assert_api_version():
    if os.environ["CORTEX_VERSION"] != consts.CORTEX_VERSION:
        raise ValueError(
            "api version mismatch (context: {}, image: {})".format(
                os.environ["CORTEX_VERSION"], consts.CORTEX_VERSION
            )
        )


def get_spec(cache_dir, s3_path):
    local_spec_path = os.path.join(cache_dir, "api_spec.msgpack")
    bucket, key = S3.deconstruct_s3_path(s3_path)
    S3(bucket, client_config={}).download_file(key, local_spec_path)
    return util.read_msgpack(local_spec_path)


def extract_waitress_params(config):
    waitress_kwargs = {}
    if config is not None:
        for key, value in config.items():
            if key.startswith("waitress_"):
                waitress_kwargs[key[len("waitress_") :]] = value

    if len(waitress_kwargs) > 0:
        cx_logger().info("waitress parameters: {}".format(waitress_kwargs))

    return waitress_kwargs


def start(args):
    assert_api_version()
    storage = S3(bucket=os.environ["CORTEX_BUCKET"], region=os.environ["AWS_REGION"])
    try:
        raw_api_spec = get_spec(args.cache_dir, args.spec)
        api = API(storage=storage, cache_dir=args.cache_dir, **raw_api_spec)
        client = api.predictor.initialize_client(args)
        cx_logger().info("loading the predictor from {}".format(api.predictor.path))
        predictor_impl = api.predictor.initialize_impl(args.project_dir, client)

        local_cache["api"] = api
        local_cache["client"] = client
        local_cache["predictor_impl"] = predictor_impl
    except:
        cx_logger().exception("failed to start api")
        sys.exit(1)

    if api.tracker is not None and api.tracker.model_type == "classification":
        try:
            local_cache["class_set"] = api.get_cached_classes()
        except Exception as e:
            cx_logger().warn("an error occurred while attempting to load classes", exc_info=True)

    waitress_kwargs = extract_waitress_params(api.predictor.config)
    waitress_kwargs["listen"] = "*:{}".format(args.port)

    cx_logger().info("{} api is live".format(api.name))
    open("/health_check.txt", "a").close()
    serve(app, **waitress_kwargs)


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--port", type=int, required=True, help="port (on localhost) to use")
    na.add_argument(
        "--tf-serve-port",
        type=int,
        required=False,
        help="port (on localhost) where tf serving runs",
    )
    na.add_argument(
        "--spec", required=True, help="s3 path to api spec (e.g. s3://bucket/path/to/api_spec.json)"
    )
    na.add_argument("--model-dir", required=False, help="directory to download the model to")
    na.add_argument("--cache-dir", required=True, help="local path for the api cache")
    na.add_argument("--project-dir", required=True, help="local path for the project zip file")
    parser.set_defaults(func=start)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
