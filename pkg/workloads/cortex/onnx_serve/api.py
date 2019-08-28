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
import builtins

from flask import Flask, request, jsonify, g
from flask_api import status
from waitress import serve
import onnxruntime as rt
import numpy as np

from cortex.lib import util, package, Context, api_utils
from cortex.lib.storage import S3
from cortex.lib.log import get_logger, print_obj
from cortex.lib.exceptions import CortexException, UserRuntimeException, UserException
from cortex.lib.stringify import truncate


def cortex_print(*args, **kwargs):
    logger.info(*args)


builtins.print = cortex_print

logger = get_logger()
logger.propagate = False  # prevent double logging (flask modifies root logger)

app = Flask(__name__)

app.json_encoder = util.json_tricks_encoder

# https://github.com/microsoft/onnxruntime/blob/v0.4.0/onnxruntime/python/onnxruntime_pybind_mlvalue.cc
ONNX_TO_NP_TYPE = {
    "tensor(float16)": "float16",
    "tensor(float)": "float32",
    "tensor(double)": "float64",
    "tensor(int32)": "int32",
    "tensor(uint32)": "uint32",
    "tensor(int8)": "int8",
    "tensor(uint8)": "uint8",
    "tensor(int16)": "int16",
    "tensor(uint16)": "uint16",
    "tensor(int64)": "int64",
    "tensor(uint64)": "uint64",
    "tensor(bool)": "bool",
    "tensor(string)": "object",
}

local_cache = {
    "ctx": None,
    "api": None,
    "sess": None,
    "input_metadata": None,
    "output_metadata": None,
    "request_handler": None,
    "class_set": set(),
}


@app.after_request
def after_request(response):
    api = local_cache["api"]
    ctx = local_cache["ctx"]

    if request.path != "/{}/{}".format(ctx.app["name"], api["name"]):
        return response

    logger.info("[%s] %s", util.now_timestamp_rfc_3339(), response.status)

    predictions = None
    if "predictions" in g:
        predictions = g.predictions
    api_utils.post_request_metrics(ctx, api, response, predictions, local_cache["class_set"])

    return response


def prediction_failed(reason):
    message = "prediction failed: {}".format(reason)
    logger.error(message)
    return message, status.HTTP_406_NOT_ACCEPTABLE


@app.route("/healthz", methods=["GET"])
def health():
    return jsonify({"ok": True})


def transform_to_numpy(input_pyobj, input_metadata):
    target_dtype = ONNX_TO_NP_TYPE[input_metadata.type]
    target_shape = input_metadata.shape

    try:
        for idx, dim in enumerate(target_shape):
            if dim is None:
                target_shape[idx] = 1

        if type(input_pyobj) is not np.ndarray:
            np_arr = np.array(input_pyobj, dtype=target_dtype)
        else:
            np_arr = input_pyobj
        np_arr = np_arr.reshape(target_shape)
        return np_arr
    except Exception as e:
        raise UserException(str(e)) from e


def convert_to_onnx_input(sample, input_metadata_list):
    input_dict = {}
    if len(input_metadata_list) == 1:
        input_metadata = input_metadata_list[0]
        if util.is_dict(sample):
            if sample.get(input_metadata.name) is None:
                raise UserException('missing key "{}"'.format(input_metadata.name))
            input_dict[input_metadata.name] = transform_to_numpy(
                sample[input_metadata.name], input_metadata
            )
        else:
            try:
                input_dict[input_metadata.name] = transform_to_numpy(sample, input_metadata)
            except CortexException as e:
                e.wrap('key "{}"'.format(input_metadata.name))
                raise
    else:
        for input_metadata in input_metadata_list:
            if not util.is_dict(input_metadata):
                expected_keys = [metadata.name for metadata in input_metadata_list]
                raise UserException(
                    "expected sample to be a dictionary with keys {}".format(
                        ", ".join('"' + key + '"' for key in expected_keys)
                    )
                )

            if sample.get(input_metadata.name) is None:
                raise UserException('missing key "{}"'.format(input_metadata.name))
            try:
                input_dict[input_metadata.name] = transform_to_numpy(sample, input_metadata)
            except CortexException as e:
                e.wrap('key "{}"'.format(input_metadata.name))
                raise
    return input_dict


@app.route("/<app_name>/<api_name>", methods=["POST"])
def predict(app_name, api_name):
    debug = request.args.get("debug").lower() == "true"

    try:
        payload = request.get_json()
    except Exception as e:
        return "Malformed JSON", status.HTTP_400_BAD_REQUEST

    sess = local_cache["sess"]
    api = local_cache["api"]
    ctx = local_cache["ctx"]
    request_handler = local_cache.get("request_handler")
    input_metadata = local_cache["input_metadata"]
    output_metadata = local_cache["output_metadata"]

    response = {}

    if not util.is_dict(payload) or "samples" not in payload:
        message = 'top level "samples" key not found in request'
        if debug:
            message += "; payload: {}".format(truncate(payload))
        return prediction_failed(message)

    predictions = []
    samples = payload["samples"]
    if not util.is_list(samples):
        return prediction_failed('expected the value of key "samples" to be a list of json objects')

    for i, sample in enumerate(payload["samples"]):
        try:
            print_obj("sample", sample, debug)

            prepared_sample = sample
            if request_handler is not None and util.has_function(request_handler, "pre_inference"):
                try:
                    prepared_sample = request_handler.pre_inference(sample, input_metadata)
                    print_obj("pre_inference", prepared_sample, debug)
                except Exception as e:
                    raise UserRuntimeException(
                        api["request_handler"], "pre_inference request handler"
                    ) from e

            inference_input = convert_to_onnx_input(prepared_sample, input_metadata)
            model_outputs = sess.run([], inference_input)
            result = []
            for model_output in model_outputs:
                if type(model_output) is np.ndarray:
                    result.append(model_output.tolist())
                else:
                    result.append(model_output)

            print_obj("inference", result, debug)

            if request_handler is not None and util.has_function(request_handler, "post_inference"):
                try:
                    result = request_handler.post_inference(result, output_metadata)
                except Exception as e:
                    raise UserRuntimeException(
                        api["request_handler"], "post_inference request handler"
                    ) from e

                print_obj("post_inference", result, debug)

        except CortexException as e:
            e.wrap("error", "sample {}".format(i + 1))
            logger.error(str(e))
            logger.exception(
                "An error occurred, see `cx logs -v api {}` for more details.".format(api["name"])
            )
            return prediction_failed(str(e))
        except Exception as e:
            logger.exception(
                "An error occurred, see `cx logs -v api {}` for more details.".format(api["name"])
            )
            return prediction_failed(str(e))

        predictions.append(result)
    g.predictions = predictions
    response["predictions"] = predictions

    return jsonify(response)


def extract_signature(metadata_list):
    metadata = {}
    for meta in metadata_list:
        numpy_type = ONNX_TO_NP_TYPE.get(meta.type, meta.type)
        metadata[meta.name] = {"shape": meta.shape, "type": numpy_type}

    return metadata


@app.route("/<app_name>/<api_name>/signature", methods=["GET"])
def get_signature(app_name, api_name):
    return jsonify({"signature": extract_signature(local_cache["input_metadata"])})


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

        bucket_name, prefix = ctx.storage.deconstruct_s3_path(api["model"])
        model_path = os.path.join(args.model_dir, os.path.basename(prefix))
        if not os.path.exists(model_path):
            s3_client = S3(bucket_name, client_config={})
            s3_client.download_file(prefix, model_path)

        if args.only_download:
            return

        if api.get("request_handler") is not None:
            package.install_packages(ctx.python_packages, ctx.storage)
            local_cache["request_handler"] = ctx.get_request_handler_impl(api["name"])

        sess = rt.InferenceSession(model_path)
        local_cache["sess"] = sess
        local_cache["input_metadata"] = sess.get_inputs()
        logger.info(
            "input_metadata: {}".format(truncate(extract_signature(local_cache["input_metadata"])))
        )
        local_cache["output_metadata"] = sess.get_outputs()
        logger.info(
            "output_metadata: {}".format(
                truncate(extract_signature(local_cache["output_metadata"]))
            )
        )
    except CortexException as e:
        e.wrap("error")
        logger.error(str(e))
        if api is not None:
            logger.exception(
                "An error occured starting the api, see `cx logs -v api {}` for more details".format(
                    api["name"]
                )
            )
        sys.exit(1)

    if api.get("tracker") is not None and api["tracker"].get("model_type") == "classification":
        try:
            local_cache["class_set"] = api_utils.get_classes(ctx, api["name"])
        except Exception as e:
            logger.warn("An error occurred while attempting to load classes", exc_info=True)

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
    na.add_argument("--model-dir", required=True, help="Directory to download the model to")
    na.add_argument("--cache-dir", required=True, help="Local path for the context cache")
    na.add_argument(
        "--only-download",
        required=False,
        help="Only download model (for init-containers)",
        default=False,
    )

    parser.set_defaults(func=start)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
