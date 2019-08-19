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
import time
from flask import Flask, request, jsonify
from flask_api import status
from waitress import serve
import grpc
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import get_model_metadata_pb2
from tensorflow_serving.apis import prediction_service_pb2_grpc
from google.protobuf import json_format

from cortex import consts
from cortex.lib import util, tf_lib, package, Context
from cortex.lib.log import get_logger
from cortex.lib.storage import S3, LocalStorage
from cortex.lib.exceptions import CortexException, UserRuntimeException, UserException
from cortex.lib.context import create_transformer_inputs_from_map

logger = get_logger()
logger.propagate = False  # prevent double logging (flask modifies root logger)

app = Flask(__name__)
app.json_encoder = util.json_tricks_encoder

local_cache = {
    "ctx": None,
    "model": None,
    "estimator": None,
    "target_col": None,
    "target_col_type": None,
    "stub": None,
    "api": None,
    "trans_impls": {},
    "required_inputs": None,
    "metadata": None,
    "target_vocab_populated": None,
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


def transform_sample(sample):
    ctx = local_cache["ctx"]
    model = local_cache["model"]

    transformed_sample = {}

    for column_name in ctx.extract_column_names(model["input"]):
        if ctx.is_raw_column(column_name):
            transformed_value = sample[column_name]
        else:
            transformed_column = ctx.transformed_columns[column_name]
            trans_impl = local_cache["trans_impls"][column_name]
            if not hasattr(trans_impl, "transform_python"):
                raise UserException(
                    "transformed column " + column_name,
                    "transformer " + transformed_column["transformer"],
                    "transform_python function is missing",
                )
            input = ctx.populate_values(
                transformed_column["input"], None, preserve_column_refs=True
            )
            transformer_input = create_transformer_inputs_from_map(input, sample)
            transformed_value = trans_impl.transform_python(transformer_input)

        transformed_sample[column_name] = transformed_value

    return transformed_sample


def create_prediction_request(transformed_sample):
    ctx = local_cache["ctx"]
    signature_def = local_cache["metadata"]["signatureDef"]
    signature_key = list(signature_def.keys())[0]
    prediction_request = predict_pb2.PredictRequest()
    prediction_request.model_spec.name = "model"
    prediction_request.model_spec.signature_name = signature_key

    for column_name, value in transformed_sample.items():
        column_type = ctx.get_inferred_column_type(column_name)
        data_type = tf_lib.CORTEX_TYPE_TO_TF_TYPE[column_type]
        shape = []
        for dim in signature_def[signature_key]["tensorShape"]["dim"]:
            shape.append(int(dim["size"]))

        try:
            tensor_proto = tf.make_tensor_proto(
                np.array(value).reshape(shape), dtype=data_type, shape=shape
            )
            prediction_request.inputs[column_name].CopyFrom(tensor_proto)
        except Exception as e:
            raise UserException(
                'key "{}"'.format(column_name), "expected shape {}".format(shape)
            ) from e

    return prediction_request


def create_raw_prediction_request(sample):
    signature_def = local_cache["metadata"]["signatureDef"]
    signature_key = list(signature_def.keys())[0]
    prediction_request = predict_pb2.PredictRequest()
    prediction_request.model_spec.name = "model"
    prediction_request.model_spec.signature_name = signature_key

    for column_name, value in sample.items():

        if util.is_list(value):
            shape = [len(value)]
            for dim in signature_def[signature_key]["inputs"][column_name]["tensorShape"]["dim"]:
                shape.append(int(dim["size"]))
        else:
            shape = [1]

        sig_type = signature_def[signature_key]["inputs"][column_name]["dtype"]

        try:
            tensor_proto = tf.make_tensor_proto(value, dtype=DTYPE_TO_TF_TYPE[sig_type])
            prediction_request.inputs[column_name].CopyFrom(tensor_proto)
        except Exception as e:
            raise UserException(
                'key "{}"'.format(column_name), "expected shape {}".format(shape)
            ) from e

    return prediction_request


def reverse_transform(value):
    ctx = local_cache["ctx"]
    model = local_cache["model"]
    target_col = local_cache["target_col"]

    trans_impl = local_cache["trans_impls"].get(target_col["name"])
    if not (trans_impl and hasattr(trans_impl, "reverse_transform_python")):
        return None

    input = ctx.populate_values(target_col["input"], None, preserve_column_refs=False)
    try:
        result = trans_impl.reverse_transform_python(value, input)
    except Exception as e:
        raise UserRuntimeException(
            "transformer " + target_col["transformer"], "function reverse_transform_python"
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
    estimator = local_cache["estimator"]
    target_col_type = local_cache["target_col_type"]

    if target_col_type in {consts.COLUMN_TYPE_STRING, consts.COLUMN_TYPE_INT}:
        prediction_key = "class_ids"
    else:
        prediction_key = "predictions"

    if estimator["prediction_key"]:
        prediction_key = estimator["prediction_key"]

    results_dict = json_format.MessageToDict(response_proto)
    outputs = results_dict["outputs"]
    value_key = DTYPE_TO_VALUE_KEY[outputs[prediction_key]["dtype"]]
    prediction = outputs[prediction_key][value_key][0]

    target_vocab_estimators = {
        "dnn_classifier",
        "linear_classifier",
        "dnn_linear_combined_classifier",
        "boosted_trees_classifier",
    }
    if (
        estimator["namespace"] == "cortex"
        and estimator["name"] in target_vocab_estimators
        and model["input"].get("target_vocab") is not None
    ):
        prediction = local_cache["target_vocab_populated"][int(prediction)]
    else:
        prediction = util.cast(prediction, target_col_type)

    result = {}
    result["prediction"] = prediction
    result["prediction_reversed"] = reverse_transform(prediction)

    result["response"] = {}
    for key in outputs.keys():
        value_key = DTYPE_TO_VALUE_KEY[outputs[key]["dtype"]]
        result["response"][key] = outputs[key][value_key]

    return result


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


def parse_response_proto_raw(response_proto):
    results_dict = json_format.MessageToDict(response_proto)
    outputs = results_dict["outputs"]

    outputs_simplified = {}
    for key in outputs.keys():
        value_key = DTYPE_TO_VALUE_KEY[outputs[key]["dtype"]]
        outputs_simplified[key] = outputs[key][value_key]

    return {"response": outputs_simplified}


def run_predict(sample):
    ctx = local_cache["ctx"]
    request_handler = local_cache.get("request_handler")

    prepared_sample = sample
    if request_handler is not None and util.has_function(request_handler, "pre_inference"):
        prepared_sample = request_handler.pre_inference(
            sample, local_cache["metadata"]["signatureDef"]
        )

    validate_sample(prepared_sample)

    if util.is_resource_ref(local_cache["api"]["model"]):
        for column in local_cache["required_inputs"]:
            column_type = ctx.get_inferred_column_type(column["name"])
            prepared_sample[column["name"]] = util.upcast(
                prepared_sample[column["name"]], column_type
            )

        transformed_sample = transform_sample(prepared_sample)

        prediction_request = create_prediction_request(transformed_sample)
        response_proto = local_cache["stub"].Predict(prediction_request, timeout=300.0)
        result = parse_response_proto(response_proto)
        result["transformed_sample"] = transformed_sample
    else:
        prediction_request = create_raw_prediction_request(prepared_sample)
        response_proto = local_cache["stub"].Predict(prediction_request, timeout=300.0)
        result = parse_response_proto_raw(response_proto)

    if request_handler is not None and util.has_function(request_handler, "post_inference"):
        result = request_handler.post_inference(result, local_cache["metadata"]["signatureDef"])

    return result


def validate_sample(sample):
    api = local_cache["api"]
    if util.is_resource_ref(api["model"]):
        ctx = local_cache["ctx"]
        for column in local_cache["required_inputs"]:
            if column["name"] not in sample:
                raise UserException('missing key "{}"'.format(column["name"]))
            sample_val = sample[column["name"]]
            column_type = ctx.get_inferred_column_type(column["name"])
            is_valid = util.CORTEX_TYPE_TO_VALIDATOR[column_type](sample_val)

            if not is_valid:
                raise UserException(
                    'key "{}"'.format(column["name"]), "expected type " + column_type
                )
    else:
        signature = extract_signature()
        for input_name, metadata in signature.items():
            if input_name not in sample:
                raise UserException('missing key "{}"'.format(input_name))


def prediction_failed(reason):
    message = "prediction failed: " + reason

    logger.error(message)
    return message, status.HTTP_406_NOT_ACCEPTABLE


@app.route("/healthz", methods=["GET"])
def health():
    return jsonify({"ok": True})


@app.route("/<deployment_name>/<api_name>", methods=["POST"])
def predict(deployment_name, api_name):

    try:
        payload = request.get_json()
    except Exception as e:
        return "Malformed JSON", status.HTTP_400_BAD_REQUEST

    ctx = local_cache["ctx"]
    api = local_cache["api"]

    response = {}

    if not util.is_dict(payload) or "samples" not in payload:
        return prediction_failed('top level "samples" key not found in request')

    predictions = []
    samples = payload["samples"]
    if not util.is_list(samples):
        return prediction_failed('expected the value of key "samples" to be a list of json objects')

    for i, sample in enumerate(payload["samples"]):
        try:
            result = run_predict(sample)
        except CortexException as e:
            e.wrap("error", "sample {}".format(i + 1))
            logger.error(str(e))
            logger.exception(
                "An error occurred, see `cortex logs -v api {}` for more details.".format(
                    api["name"]
                )
            )
            return prediction_failed(str(e))
        except Exception as e:
            logger.exception(
                "An error occurred, see `cortex logs -v api {}` for more details.".format(
                    api["name"]
                )
            )
            return prediction_failed(str(e))

        predictions.append(result)

    response["predictions"] = predictions
    response["resource_id"] = api["id"]

    return jsonify(response)


def extract_signature():
    signature_def = local_cache["metadata"]["signatureDef"]
    if signature_def.get("predict") is None or signature_def["predict"].get("inputs") is None:
        raise UserException('unable to find "predict" in model\'s signature definition')

    metadata = {}
    for input_name, input_metadata in signature_def["predict"]["inputs"].items():
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
        metadata = extract_signature()
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


def download_dir_external(ctx, s3_path, local_path):
    util.mkdir_p(local_path)
    bucket_name, prefix = ctx.storage.deconstruct_s3_path(s3_path)
    storage_client = S3(bucket_name, client_config={})
    objects = [obj[len(prefix) + 1:] for obj in storage_client.search(prefix=prefix)]
    version = prefix.rstrip("/").split("/")[-1]
    local_path = os.path.join(local_path, version)
    for obj in objects:
        if not os.path.exists(os.path.dirname(obj)):
            util.mkdir_p(os.path.join(local_path, os.path.dirname(obj)))

        ctx.storage.download_file_external(
            bucket_name + "/" + os.path.join(prefix, obj), os.path.join(local_path, obj)
        )


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


@app.after_request
def after_request(response):
    if request.full_path.startswith("/healthz"):
        return response
    logger.info("[%s] %s", util.now_timestamp_rfc_3339(), response.status)
    return response


@app.errorhandler(Exception)
def exceptions(e):
    logger.exception(e)
    return jsonify(error=str(e)), 500


def start(args):
    ctx = Context(s3_path=args.context, cache_dir=args.cache_dir, workload_id=args.workload_id)

    api = ctx.apis_id_map[args.api]
    local_cache["api"] = api
    local_cache["ctx"] = ctx

    try:
        if api.get("request_handler") is not None or util.is_resource_ref(api["model"]):
            package.install_packages(ctx.python_packages, ctx.storage)

        if api.get("request_handler") is not None:
            local_cache["request_handler"] = ctx.get_request_handler_impl(api["name"])
        
        if not os.path.isdir(args.model_dir):
            if util.is_resource_ref(api["model"]):
                model_name = util.get_resource_ref(api["model"])
                model = ctx.models[model_name]
                ctx.storage.download_and_unzip(model["key"], args.model_dir)
            else:
                download_dir_external(ctx, api["model"], args.model_dir)

        if args.only_download:
            return
            
        if util.is_resource_ref(api["model"]):
            model_name = util.get_resource_ref(api["model"])
            model = ctx.models[model_name]
            estimator = ctx.estimators[model["estimator"]]

            local_cache["model"] = model
            local_cache["estimator"] = estimator
            local_cache["target_col"] = ctx.columns[util.get_resource_ref(model["target_column"])]
            local_cache["target_col_type"] = ctx.get_inferred_column_type(
                util.get_resource_ref(model["target_column"])
            )

            log_level = "DEBUG"
            if ctx.environment is not None and ctx.environment.get("log_level") is not None:
                log_level = ctx.environment["log_level"].get("tensorflow", "DEBUG")
            tf_lib.set_logging_verbosity(log_level)

            for column_name in ctx.extract_column_names([model["input"], model["target_column"]]):
                if ctx.is_transformed_column(column_name):
                    trans_impl, _ = ctx.get_transformer_impl(column_name)
                    local_cache["trans_impls"][column_name] = trans_impl
                    transformed_column = ctx.transformed_columns[column_name]

                    # cache aggregate values
                    for resource_name in util.extract_resource_refs(transformed_column["input"]):
                        if resource_name in ctx.aggregates:
                            ctx.get_obj(ctx.aggregates[resource_name]["key"])

            local_cache["required_inputs"] = tf_lib.get_base_input_columns(model["name"], ctx)

            if util.is_dict(model["input"]) and model["input"].get("target_vocab") is not None:
                local_cache["target_vocab_populated"] = ctx.populate_values(
                    model["input"]["target_vocab"], None, False
                )
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
