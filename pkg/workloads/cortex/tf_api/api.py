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
import builtins

import tensorflow as tf
from flask import Flask, request, jsonify, g
from flask_api import status
from waitress import serve
import grpc
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import get_model_metadata_pb2
from tensorflow_serving.apis import prediction_service_pb2_grpc
from google.protobuf import json_format

from cortex.lib import util, package, Context, api_utils
from cortex.lib.storage import S3
from cortex.lib.log import get_logger, debug_obj
from cortex.lib.exceptions import CortexException, UserRuntimeException, UserException
from cortex.lib.stringify import truncate


def cortex_print(*args, **kwargs):
    logger.info(*args)


builtins.print = cortex_print

logger = get_logger()
logger.propagate = False  # prevent double logging (flask modifies root logger)

app = Flask(__name__)
app.json_encoder = util.json_tricks_encoder


local_cache = {
    "ctx": None,
    "stub": None,
    "api": None,
    "metadata": None,
    "request_handler": None,
    "class_set": set(),
}


DTYPE_TO_VALUE_KEY = {
    "DT_INT32": "intVal",
    "DT_INT64": "int64Val",
    "DT_FLOAT": "floatVal",
    "DT_STRING": "stringVal",
    "DT_BOOL": "boolVal",
    "DT_DOUBLE": "doubleVal",
    "DT_HALF": "halfVal",
    "DT_COMPLEX64": "scomplexVal",
    "DT_COMPLEX128": "dcomplexVal",
}

DTYPE_TO_TF_TYPE = {
    "DT_FLOAT": tf.float32,
    "DT_DOUBLE": tf.float64,
    "DT_INT32": tf.int32,
    "DT_UINT8": tf.uint8,
    "DT_INT16": tf.int16,
    "DT_INT8": tf.int8,
    "DT_STRING": tf.string,
    "DT_COMPLEX64": tf.complex64,
    "DT_INT64": tf.int64,
    "DT_BOOL": tf.bool,
    "DT_QINT8": tf.qint8,
    "DT_QUINT8": tf.quint8,
    "DT_QINT32": tf.qint32,
    "DT_BFLOAT16": tf.bfloat16,
    "DT_QINT16": tf.qint16,
    "DT_QUINT16": tf.quint16,
    "DT_UINT16": tf.uint16,
    "DT_COMPLEX128": tf.complex128,
    "DT_HALF": tf.float16,
    "DT_RESOURCE": tf.resource,
    "DT_VARIANT": tf.variant,
    "DT_UINT32": tf.uint32,
    "DT_UINT64": tf.uint64,
}


@app.after_request
def after_request(response):
    api = local_cache["api"]
    ctx = local_cache["ctx"]

    if request.path != "/{}/{}".format(ctx.app["name"], api["name"]):
        return response

    logger.info("[%s] %s", util.now_timestamp_rfc_3339(), response.status)

    prediction = None
    if "prediction" in g:
        prediction = g.prediction
    api_utils.post_request_metrics(ctx, api, response, prediction, local_cache["class_set"])

    return response


def create_prediction_request(sample):
    signature_def = local_cache["metadata"]["signatureDef"]
    signature_key = local_cache["api"]["tf_serving"]["signature_key"]
    prediction_request = predict_pb2.PredictRequest()
    prediction_request.model_spec.name = "model"
    prediction_request.model_spec.signature_name = signature_key

    for column_name, value in sample.items():
        shape = []
        for dim in signature_def[signature_key]["inputs"][column_name]["tensorShape"]["dim"]:
            shape.append(int(dim["size"]))

        sig_type = signature_def[signature_key]["inputs"][column_name]["dtype"]

        try:
            tensor_proto = tf.compat.v1.make_tensor_proto(value, dtype=DTYPE_TO_TF_TYPE[sig_type])
            prediction_request.inputs[column_name].CopyFrom(tensor_proto)
        except Exception as e:
            raise UserException(
                'key "{}"'.format(column_name), "expected shape {}".format(shape)
            ) from e

    return prediction_request


def create_get_model_metadata_request():
    get_model_metadata_request = get_model_metadata_pb2.GetModelMetadataRequest()
    get_model_metadata_request.model_spec.name = "model"
    get_model_metadata_request.metadata_field.append("signature_def")
    return get_model_metadata_request


def run_get_model_metadata():
    request = create_get_model_metadata_request()
    resp = local_cache["stub"].GetModelMetadata(request, timeout=10.0)
    sigAny = resp.metadata["signature_def"]
    signature_def_map = get_model_metadata_pb2.SignatureDefMap()
    sigAny.Unpack(signature_def_map)
    sigmap = json_format.MessageToDict(signature_def_map)
    return sigmap


def parse_response_proto(response_proto):
    results_dict = json_format.MessageToDict(response_proto)
    outputs = results_dict["outputs"]

    outputs_simplified = {}
    for key in outputs.keys():
        value_key = DTYPE_TO_VALUE_KEY[outputs[key]["dtype"]]
        outputs_simplified[key] = outputs[key][value_key]

    return {"response": outputs_simplified}


def run_predict(sample, debug=False):
    ctx = local_cache["ctx"]
    api = local_cache["api"]
    request_handler = local_cache.get("request_handler")

    prepared_sample = sample

    debug_obj("sample", sample, debug)
    if request_handler is not None and util.has_function(request_handler, "pre_inference"):
        try:
            prepared_sample = request_handler.pre_inference(
                sample, local_cache["metadata"]["signatureDef"]
            )
            debug_obj("pre_inference", prepared_sample, debug)
        except Exception as e:
            raise UserRuntimeException(
                api["request_handler"], "pre_inference request handler"
            ) from e

    validate_sample(prepared_sample)

    prediction_request = create_prediction_request(prepared_sample)
    response_proto = local_cache["stub"].Predict(prediction_request, timeout=300.0)
    result = parse_response_proto(response_proto)
    debug_obj("inference", result, debug)

    if request_handler is not None and util.has_function(request_handler, "post_inference"):
        try:
            result = request_handler.post_inference(result, local_cache["metadata"]["signatureDef"])
            debug_obj("post_inference", result, debug)
        except Exception as e:
            raise UserRuntimeException(
                api["request_handler"], "post_inference request handler"
            ) from e

    return result


def validate_sample(sample):
    signature = extract_signature(
        local_cache["metadata"]["signatureDef"], local_cache["api"]["tf_serving"]["signature_key"]
    )
    for input_name, _ in signature.items():
        if input_name not in sample:
            raise UserException('missing key "{}"'.format(input_name))


def prediction_failed(reason):
    message = "prediction failed: {}".format(reason)
    logger.error(message)
    return message, status.HTTP_406_NOT_ACCEPTABLE


@app.route("/healthz", methods=["GET"])
def health():
    return jsonify({"ok": True})


@app.route("/<deployment_name>/<api_name>", methods=["POST"])
def predict(deployment_name, api_name):
    debug = (
        request.args.get("debug", "false") is not None
        and request.args.get("debug").lower() == "true"
    )

    try:
        sample = request.get_json()
    except Exception as e:
        return "Malformed JSON", status.HTTP_400_BAD_REQUEST

    ctx = local_cache["ctx"]
    api = local_cache["api"]

    response = {}

    try:
        result = run_predict(sample, debug)
    except CortexException as e:
        e.wrap("error")
        logger.error(str(e))
        logger.exception(
            "An error occurred, see `cortex logs -v api {}` for more details.".format(api["name"])
        )
        return prediction_failed(str(e))
    except Exception as e:
        logger.exception(
            "An error occurred, see `cortex logs -v api {}` for more details.".format(api["name"])
        )
        return prediction_failed(str(e))

    g.prediction = result

    return jsonify(result)


def extract_signature(signature_def, signature_key):
    if (
        signature_def.get(signature_key) is None
        or signature_def[signature_key].get("inputs") is None
    ):
        raise UserException(
            'unable to find "' + signature_key + "\" in model's signature definition"
        )

    metadata = {}
    for input_name, input_metadata in signature_def[signature_key]["inputs"].items():
        metadata[input_name] = {
            "shape": [int(dim["size"]) for dim in input_metadata["tensorShape"]["dim"]],
            "type": DTYPE_TO_TF_TYPE[input_metadata["dtype"]].name,
        }
    return metadata


@app.route("/<app_name>/<api_name>/signature", methods=["GET"])
def get_signature(app_name, api_name):
    ctx = local_cache["ctx"]
    api = local_cache["api"]

    try:
        metadata = extract_signature(
            local_cache["metadata"]["signatureDef"],
            local_cache["api"]["tf_serving"]["signature_key"],
        )
    except CortexException as e:
        logger.error(str(e))
        logger.exception(
            "An error occurred, see `cortex logs -v api {}` for more details.".format(api["name"])
        )
        return str(e), HTTP_404_NOT_FOUND
    except Exception as e:
        logger.exception(
            "An error occurred, see `cortex logs -v api {}` for more details.".format(api["name"])
        )
        return str(e), HTTP_404_NOT_FOUND

    response = {"signature": metadata}
    return jsonify(response)


def validate_model_dir(model_dir):
    """
    validates that model_dir has the expected directory tree.

    For example (your TF serving version number may be different):

    1562353043/
        saved_model.pb
        variables/
            variables.data-00000-of-00001
            variables.index
    """
    version = os.listdir(model_dir)[0]
    if not version.isdigit():
        raise UserException(
            "No versions of servable default found under base path in model_dir. See docs.cortex.dev for how to properly package your TensorFlow model"
        )

    if "saved_model.pb" not in os.listdir(os.path.join(model_dir, version)):
        raise UserException(
            'Expected packaged model to have a "saved_model.pb" file. See docs.cortex.dev for how to properly package your TensorFlow model'
        )


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

        if api.get("request_handler") is not None:
            package.install_packages(ctx.python_packages, ctx.storage)
            local_cache["request_handler"] = ctx.get_request_handler_impl(api["name"])
    except CortexException as e:
        e.wrap("error")
        logger.error(str(e))
        logger.exception(
            "An error occurred, see `cortex logs -v api {}` for more details.".format(api["name"])
        )
        sys.exit(1)
    except Exception as e:
        logger.exception(
            "An error occurred, see `cortex logs -v api {}` for more details.".format(api["name"])
        )
        sys.exit(1)

    try:
        validate_model_dir(args.model_dir)
    except Exception as e:
        logger.exception(e)
        sys.exit(1)

    if api.get("tracker") is not None and api["tracker"].get("model_type") == "classification":
        try:
            local_cache["class_set"] = api_utils.get_classes(ctx, api["name"])
        except Exception as e:
            logger.warn("An error occurred while attempting to load classes", exc_info=True)

    channel = grpc.insecure_channel("localhost:" + str(args.tf_serve_port))
    local_cache["stub"] = prediction_service_pb2_grpc.PredictionServiceStub(channel)

    # wait a bit for tf serving to start before querying metadata
    limit = 300
    for i in range(limit):
        try:
            local_cache["metadata"] = run_get_model_metadata()
            break
        except Exception as e:
            if i == limit - 1:
                logger.exception(
                    "An error occurred, see `cortex logs -v api {}` for more details.".format(
                        api["name"]
                    )
                )
                sys.exit(1)

        time.sleep(1)
    logger.info(
        "model_signature: {}".format(
            extract_signature(
                local_cache["metadata"]["signatureDef"],
                local_cache["api"]["tf_serving"]["signature_key"],
            )
        )
    )
    serve(app, listen="*:{}".format(args.port))


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--workload-id", required=True, help="Workload ID")
    na.add_argument("--port", type=int, required=True, help="Port (on localhost) to use")
    na.add_argument(
        "--tf-serve-port", type=int, required=True, help="Port (on localhost) where TF Serving runs"
    )
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
