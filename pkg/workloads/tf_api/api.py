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
import tensorflow as tf
import traceback
from flask import Flask, request, jsonify
from flask_api import status
from waitress import serve
from grpc.beta import implementations
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import get_model_metadata_pb2
from tensorflow_serving.apis import prediction_service_pb2
from lib import util, tf_lib, package, Context
from lib.log import get_logger
from lib.exceptions import CortexException, UserRuntimeException, UserException
from google.protobuf import json_format
import time

logger = get_logger()
logger.propagate = False  # prevent double logging (flask modifies root logger)

app = Flask(__name__)

local_cache = {
    "ctx": None,
    "model": None,
    "stub": None,
    "api": None,
    "trans_impls": {},
    "transform_args_cache": {},
    "required_inputs": None,
    "metadata": None,
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


def transform_sample(sample):
    ctx = local_cache["ctx"]
    model = local_cache["model"]

    transformed_sample = {}

    for column_name in model["feature_columns"]:
        if ctx.is_raw_column(column_name):
            transformed_value = sample[column_name]
        else:
            inputs = ctx.create_column_inputs_map(sample, column_name)
            trans_impl = local_cache["trans_impls"][column_name]
            if not hasattr(trans_impl, "transform_python"):
                raise UserException(
                    "transformed column " + column_name,
                    "transformer " + ctx.transformed_sample[column_name]["transformer"],
                    "transform_python function missing",
                )

            args = local_cache["transform_args_cache"].get(column_name, {})
            transformed_value = trans_impl.transform_python(inputs, args)
        transformed_sample[column_name] = transformed_value

    return transformed_sample


def create_prediction_request(transformed_sample):
    ctx = local_cache["ctx"]
    signatureDef = local_cache["metadata"]["signatureDef"]
    signature_key = list(signatureDef.keys())[0]
    prediction_request = predict_pb2.PredictRequest()
    prediction_request.model_spec.name = "default"
    prediction_request.model_spec.signature_name = signature_key

    for column_name, value in transformed_sample.items():
        column_Type = ctx.get_inferred_column_type(column_name)
        data_type = tf_lib.CORTEX_TYPE_TO_TF_TYPE[column_Type]
        shape = [1]
        if util.is_list(value):
            shape = [len(value)]
        tensor_proto = tf.make_tensor_proto([value], dtype=data_type, shape=shape)
        prediction_request.inputs[column_name].CopyFrom(tensor_proto)

    return prediction_request


def reverse_transform(value):
    ctx = local_cache["ctx"]
    model = local_cache["model"]

    trans_impl = local_cache["trans_impls"].get(model["target_column"], None)
    if not (trans_impl and hasattr(trans_impl, "reverse_transform_python")):
        return None

    transformer_name = model["target_column"]
    input_schema = ctx.transformed_columns[transformer_name]["inputs"]

    if input_schema.get("args", None) is not None and len(input_schema["args"]) > 0:
        args = local_cache["transform_args_cache"].get(transformer_name, {})
    try:
        result = trans_impl.reverse_transform_python(value, args)
    except Exception as e:
        raise UserRuntimeException(
            "transformer " + ctx.transformed_columns[model["target_column"]]["transformer"],
            "function reverse_transform_python",
        ) from e

    return result


def parse_response_proto(response_proto):
    """
    response_proto is type tensorflow_serving.apis.predict_pb2.PredictResponse

    https://developers.google.com/protocol-buffers/docs/reference/python-generated
    https://github.com/tensorflow/serving/blob/master/tensorflow_serving/apis/predict.proto
    Also see GRPC docs
    response_proto.result() may be necessary (TF > 1.2?)
    """
    model = local_cache["model"]

    if model["type"] == "regression":
        prediction_key = "predictions"
    if model["type"] == "classification":
        prediction_key = "class_ids"

    if model["prediction_key"]:
        prediction_key = model["prediction_key"]

    results_dict = json_format.MessageToDict(response_proto)
    outputs = results_dict["outputs"]
    value_key = DTYPE_TO_VALUE_KEY[outputs[prediction_key]["dtype"]]
    predicted = outputs[prediction_key][value_key][0]

    result = {}
    for key in outputs.keys():
        value_key = DTYPE_TO_VALUE_KEY[outputs[key]["dtype"]]
        result[key] = outputs[key][value_key]

    if model["type"] == "regression":
        predicted = float(predicted)
        result["predicted_value"] = predicted
        result["predicted_value_reversed"] = reverse_transform(predicted)
    if model["type"] == "classification":
        predicted = int(predicted)
        result["predicted_class"] = predicted
        result["predicted_class_reversed"] = reverse_transform(predicted)

    return result


def create_get_model_metadata_request():
    get_model_metadata_request = get_model_metadata_pb2.GetModelMetadataRequest()
    get_model_metadata_request.model_spec.name = "default"
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


def run_predict(sample):
    transformed_sample = transform_sample(sample)
    prediction_request = create_prediction_request(transformed_sample)
    response_proto = local_cache["stub"].Predict(prediction_request, timeout=10.0)
    result = parse_response_proto(response_proto)
    util.log_indent("Raw sample:", indent=4)
    util.log_pretty(sample, indent=6)
    util.log_indent("Transformed sample:", indent=4)
    util.log_pretty(transformed_sample, indent=6)
    util.log_indent("Prediction:", indent=4)
    util.log_pretty(result, indent=6)

    return result


def is_valid_sample(sample):
    for column in local_cache["required_inputs"]:
        if column["name"] not in sample:
            return False, "{} is missing".format(column["name"])

        sample_val = sample[column["name"]]
        is_valid = util.CORTEX_TYPE_TO_UPCAST_VALIDATOR[column["type"]](sample_val)

        if not is_valid:
            return (False, "{} should be a {}".format(column["name"], column["type"]))

    return True, None


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

    model = local_cache["model"]
    api = local_cache["api"]

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

        is_valid, reason = is_valid_sample(sample)
        if not is_valid:
            return prediction_failed(sample, reason)

        for column in local_cache["required_inputs"]:
            sample[column["name"]] = util.upcast(sample[column["name"]], column["type"])

        try:
            result = run_predict(sample)
        except CortexException as e:
            e.wrap("error", "sample {}".format(i + 1))
            logger.error(str(e))
            logger.exception(
                "An error occurred, see `cx logs api {}` for more details.".format(api["name"])
            )
            return prediction_failed(sample, str(e))
        except Exception as e:
            logger.exception(
                "An error occurred, see `cx logs api {}` for more details.".format(api["name"])
            )
            return prediction_failed(sample, str(e))

        predictions.append(result)

    if model["type"] == "regression":
        response["regression_predictions"] = predictions
    if model["type"] == "classification":
        response["classification_predictions"] = predictions

    response["resource_id"] = api["id"]

    return jsonify(response)


def start(args):
    ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)
    package.install_packages(ctx.python_packages, ctx.storage)

    api = ctx.apis_id_map[args.api]
    model = ctx.models[api["model_name"]]
    tf_lib.set_logging_verbosity(ctx.environment["log_level"]["tensorflow"])

    local_cache["ctx"] = ctx
    local_cache["api"] = api
    local_cache["model"] = model

    if not os.path.isdir(args.model_dir):
        ctx.storage.download_and_unzip(model["key"], args.model_dir)

    for column_name in model["feature_columns"] + [model["target_column"]]:
        if ctx.is_transformed_column(column_name):
            trans_impl, _ = ctx.get_transformer_impl(column_name)
            local_cache["trans_impls"][column_name] = trans_impl
            transformed_column = ctx.transformed_columns[column_name]
            input_args_schema = transformed_column["inputs"]["args"]
            # cache aggregates and constants in memory
            if input_args_schema is not None:
                local_cache["transform_args_cache"][column_name] = ctx.populate_args(
                    input_args_schema
                )

    channel = implementations.insecure_channel("localhost", args.tf_serve_port)
    local_cache["stub"] = prediction_service_pb2.beta_create_PredictionService_stub(channel)

    local_cache["required_inputs"] = tf_lib.get_base_input_columns(model["name"], ctx)

    # wait a bit for tf serving to start before querying metadata
    limit = 600
    for i in range(limit):
        try:
            local_cache["metadata"] = run_get_model_metadata()
            break
        except Exception as e:
            if i == limit - 1:
                logger.exception(
                    "An error occurred, see `cx logs api {}` for more details.".format(api["name"])
                )
                sys.exit(1)

        time.sleep(1)

    logger.info("Serving model: {}".format(model["name"]))
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
