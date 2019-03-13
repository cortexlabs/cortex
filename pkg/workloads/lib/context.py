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
import json
import imp
import inspect
import importlib
from datetime import datetime
from copy import deepcopy

import consts
from lib import util, aws
from lib.exceptions import CortexException, UserException
from botocore.exceptions import ClientError
from lib.resources import ResourceMap
from lib.log import get_logger

logger = get_logger()


class Context:
    def __init__(self, **kwargs):
        if "cache_dir" in kwargs:
            self.cache_dir = kwargs["cache_dir"]
        elif "local_path" in kwargs:
            local_path_dir = os.path.dirname(os.path.abspath(kwargs["local_path"]))
            self.cache_dir = os.path.join(local_path_dir, "cache")
        else:
            raise ValueError("cache_dir must be specified (or inferred from local_path)")
        util.mkdir_p(self.cache_dir)

        if "local_path" in kwargs:
            ctx_raw = util.read_msgpack(kwargs["local_path"])
            self.ctx = _deserialize_raw_ctx(ctx_raw)
        elif "obj" in kwargs:
            self.ctx = kwargs["obj"]
        elif "s3_path":
            local_ctx_path = os.path.join(self.cache_dir, "context.json")
            bucket, key = aws.deconstruct_s3_path(kwargs["s3_path"])
            aws.download_file_from_s3(key, local_ctx_path, bucket)
            ctx_raw = util.read_msgpack(local_ctx_path)
            self.ctx = _deserialize_raw_ctx(ctx_raw)
        else:
            raise ValueError("invalid context args: " + kwargs)

        self.workload_id = kwargs.get("workload_id")

        self.id = self.ctx["id"]
        self.key = self.ctx["key"]
        self.cortex_config = self.ctx["cortex_config"]
        self.dataset_version = self.ctx["dataset_version"]
        self.root = self.ctx["root"]
        self.raw_dataset = self.ctx["raw_dataset"]
        self.status_prefix = self.ctx["status_prefix"]
        self.app = self.ctx["app"]
        self.environment = self.ctx["environment"]
        self.python_packages = self.ctx["python_packages"]
        self.raw_columns = self.ctx["raw_columns"]
        self.transformed_columns = self.ctx["transformed_columns"]
        self.transformers = self.ctx["transformers"]
        self.aggregators = self.ctx["aggregators"]
        self.aggregates = self.ctx["aggregates"]
        self.constants = self.ctx["constants"]
        self.models = self.ctx["models"]
        self.apis = self.ctx["apis"]
        self.training_datasets = {k: v["dataset"] for k, v in self.models.items()}

        self.bucket = self.cortex_config["bucket"]
        self.region = self.cortex_config["region"]
        self.api_version = self.cortex_config["api_version"]

        if self.api_version != consts.CORTEX_VERSION:
            raise ValueError(
                "API version mismatch (Context: {}, Image: {})".format(
                    self.api_version, consts.CORTEX_VERSION
                )
            )

        self.columns = util.merge_dicts_overwrite(
            self.raw_columns, self.transformed_columns  # self.aggregates
        )

        self.values = util.merge_dicts_overwrite(self.aggregates, self.constants)

        self.raw_column_names = list(self.raw_columns.keys())
        self.transformed_column_names = list(self.transformed_columns.keys())
        self.column_names = list(self.columns.keys())

        # Internal caches
        self._transformer_impls = {}
        self._aggregator_impls = {}
        self._model_impls = {}

        # This affects Tensorflow S3 access
        os.environ["AWS_REGION"] = self.region

        # Id map
        self.pp_id_map = ResourceMap(self.python_packages)
        self.rf_id_map = ResourceMap(self.raw_columns)
        self.ag_id_map = ResourceMap(self.aggregates)
        self.tf_id_map = ResourceMap(self.transformed_columns)
        self.td_id_map = ResourceMap(self.training_datasets)
        self.models_id_map = ResourceMap(self.models)
        self.apis_id_map = ResourceMap(self.apis)
        self.constants_id_map = ResourceMap(self.constants)

        self.id_map = util.merge_dicts_overwrite(
            self.pp_id_map,
            self.rf_id_map,
            self.ag_id_map,
            self.tf_id_map,
            self.td_id_map,
            self.models_id_map,
            self.apis_id_map,
            self.constants_id_map,
        )

    def is_raw_column(self, name):
        return name in self.raw_columns

    def is_transformed_column(self, name):
        return name in self.transformed_columns

    def is_constant(self, name):
        return name in self.constants

    def is_aggregate(self, name):
        return name in self.aggregates

    def create_column_inputs_map(self, values_map, column_name):
        """Construct an inputs dict with actual data"""
        columns_input_config = self.transformed_columns[column_name]["inputs"]["columns"]
        return create_inputs_map(values_map, columns_input_config)

    def get_file(self, impl_key, cache_impl_path):
        if not os.path.isfile(cache_impl_path):
            aws.download_file_from_s3(impl_key, cache_impl_path, self.bucket)
        return cache_impl_path

    def get_python_file(self, impl_key, module_name):
        cache_impl_path = os.path.join(self.cache_dir, "{}.py".format(module_name))
        self.get_file(impl_key, cache_impl_path)
        return cache_impl_path

    def get_obj(self, key):
        cache_path = os.path.join(self.cache_dir, key)
        self.get_file(key, cache_path)

        return util.read_msgpack(cache_path)

    def populate_args(self, args_dict):
        return {
            arg_name: self.get_obj(self.values[value_name]["key"])
            for arg_name, value_name in args_dict.items()
        }

    def store_aggregate_result(self, result, aggregate):
        aws.upload_msgpack_to_s3(result, aggregate["key"], self.bucket)

    def load_module(self, module_prefix, module_name, impl_key):
        full_module_name = "{}_{}".format(module_prefix, module_name)

        try:
            impl_path = self.get_python_file(impl_key, full_module_name)
        except CortexException as e:
            e.wrap("unable to find python file " + module_name)
            raise

        try:
            impl = imp.load_source(full_module_name, impl_path)
        except Exception as e:
            raise UserException("unable to load python module " + module_name) from e

        return impl, impl_path

    def get_aggregator_impl(self, column_name):
        aggregator_name = self.aggregates[column_name]["aggregator"]
        if aggregator_name in self._aggregator_impls:
            return self._aggregator_impls[aggregator_name]
        aggregator = self.aggregators[aggregator_name]

        module_prefix = "aggregator"
        if "namespace" in aggregator and aggregator.get("namespace", None) is not None:
            module_prefix += "_" + aggregator["namespace"]

        try:
            impl, impl_path = self.load_module(
                module_prefix, aggregator["name"], aggregator["impl_key"]
            )
        except CortexException as e:
            e.wrap("aggregate " + column_name, "aggregator")
            raise

        try:
            _validate_impl(impl, AGGREGATOR_IMPL_VALIDATION)
        except CortexException as e:
            e.wrap("aggregate " + column_name, "aggregator " + aggregator["name"])
            raise

        self._aggregator_impls[aggregator_name] = (impl, impl_path)
        return (impl, impl_path)

    def get_transformer_impl(self, column_name):
        if self.is_transformed_column(column_name) is not True:
            return None, None

        transformer_name = self.transformed_columns[column_name]["transformer"]

        if transformer_name in self._transformer_impls:
            return self._transformer_impls[transformer_name]

        transformer = self.transformers[transformer_name]

        module_prefix = "transformer"
        if "namespace" in transformer and transformer.get("namespace", None) is not None:
            module_prefix += "_" + transformer["namespace"]

        try:
            impl, impl_path = self.load_module(
                module_prefix, transformer["name"], transformer["impl_key"]
            )
        except CortexException as e:
            e.wrap("transformed column " + column_name, "transformer")
            raise

        try:
            _validate_impl(impl, TRANSFORMER_IMPL_VALIDATION)
        except CortexException as e:
            e.wrap("transformed column " + column_name, "transformer " + transformer["name"])
            raise

        self._transformer_impls[transformer_name] = (impl, impl_path)
        return (impl, impl_path)

    def get_model_impl(self, model_name):
        if model_name in self._model_impls:
            return self._model_impls[model_name]

        model = self.models[model_name]

        try:
            impl, impl_path = self.load_module("model", model_name, model["impl_key"])
            _validate_impl(impl, MODEL_IMPL_VALIDATION)
        except CortexException as e:
            e.wrap("model " + model_name)
            raise

        self._model_impls[model_name] = impl
        return impl

    # Mode must be "training" or "evaluation"
    def get_training_data_parts(self, model_name, mode, part_prefix="part"):
        training_dataset = self.models[model_name]["dataset"]
        if mode == "training":
            data_key = training_dataset["train_key"]
        elif mode == "evaluation":
            data_key = training_dataset["eval_key"]
        else:
            raise CortexException(
                "unrecognized training/evaluation mode {} must be one of (train_key, eval_key)".format(
                    mode
                )
            )

        training_data_parts_prefix = os.path.join(data_key, part_prefix)
        return aws.get_matching_s3_keys(self.bucket, prefix=training_data_parts_prefix)

    def column_config(self, column_name):
        if self.is_raw_column(column_name):
            return self.raw_column_config(column_name)
        elif self.is_transformed_column(column_name):
            return self.transformed_column_config(column_name)
        return None

    def raw_column_config(self, column_name):
        raw_column = self.raw_columns[column_name]
        if raw_column is None:
            return None
        config = deepcopy(raw_column)
        config_keys = ["name", "type", "required", "min", "max", "values", "tags"]
        util.keep_dict_keys(config, config_keys)
        return config

    def transformed_column_config(self, column_name):
        transformed_column = self.transformed_columns[column_name]
        if transformed_column is None:
            return None
        config = deepcopy(transformed_column)
        config_keys = ["name", "transformer", "inputs", "tags", "type"]
        util.keep_dict_keys(config, config_keys)
        config["inputs"] = self._expand_inputs_config(config["inputs"])
        config["transformer"] = self.transformer_config(config["transformer"])
        return config

    def value_config(self, value_name):
        if self.is_constant(value_name):
            return self.constant_config(value_name)
        elif self.is_aggregate(value_name):
            return self.aggregate_config(value_name)
        return None

    def constant_config(self, constant_name):
        constant = self.constants[constant_name]
        if constant is None:
            return None
        config = deepcopy(constant)
        config_keys = ["name", "type", "tags"]
        util.keep_dict_keys(config, config_keys)
        return config

    def aggregate_config(self, aggregate_name):
        aggregate = self.aggregates[aggregate_name]
        if aggregate is None:
            return None
        config = deepcopy(aggregate)
        config_keys = ["name", "type", "inputs", "aggregator", "tags"]
        util.keep_dict_keys(config, config_keys)
        config["inputs"] = self._expand_inputs_config(config["inputs"])
        config["aggregator"] = self.aggregator_config(config["aggregator"])
        return config

    def model_config(self, model_name):
        model = self.models[model_name]
        if model is None:
            return None
        model_config = deepcopy(model)
        config_keys = [
            "name",
            "type",
            "path",
            "target_column",
            "prediction_key",
            "feature_columns",
            "training_columns",
            "hparams",
            "data_partition_ratio",
            "aggregates",
            "training",
            "evaluation",
            "tags",
        ]
        util.keep_dict_keys(model_config, config_keys)

        for i, column_name in enumerate(model_config["feature_columns"]):
            model_config["feature_columns"][i] = self.column_config(column_name)

        for i, column_name in enumerate(model_config["training_columns"]):
            model_config["training_columns"][i] = self.column_config(column_name)

        model_config["target_column"] = self.column_config(model_config["target_column"])

        aggregates_dict = {key: key for key in model_config["aggregates"]}
        model_config["aggregates"] = self.populate_args(aggregates_dict)

        return model_config

    def aggregator_config(self, aggregator_name):
        aggregator = self.aggregators[aggregator_name]
        if aggregator is None:
            return None
        config = deepcopy(aggregator)
        config_keys = ["name", "output_type", "inputs"]
        util.keep_dict_keys(config, config_keys)
        config["name"] = aggregator_name  # Use the fully qualified name (includes namespace)
        return config

    def transformer_config(self, transformer_name):
        transformer = self.transformers[transformer_name]
        if transformer is None:
            return None
        config = deepcopy(transformer)
        config_keys = ["name", "output_type", "inputs"]
        util.keep_dict_keys(config, config_keys)
        config["name"] = transformer_name  # Use the fully qualified name (includes namespace)
        return config

    def _expand_inputs_config(self, inputs_config):
        inputs_config["columns"] = self._expand_columns_input_dict(inputs_config["columns"])
        inputs_config["args"] = self._expand_args_dict(inputs_config["args"])
        return inputs_config

    def _expand_columns_input_dict(self, input_columns_dict):
        expanded = {}
        for column_name, value in input_columns_dict.items():
            if util.is_str(value):
                expanded[column_name] = self.column_config(value)
            elif util.is_list(value):
                expanded[column_name] = [self.column_config(name) for name in value]
        return expanded

    def _expand_args_dict(self, args_dict):
        expanded = {}
        for arg_name, value_name in args_dict.items():
            expanded[arg_name] = self.value_config(value_name)
        return expanded

    def get_resource_status(self, resource):
        key = self.resource_status_key(resource)
        return aws.read_json_from_s3(key, self.bucket)

    def upload_resource_status_start(self, *resources):
        timestamp = util.now_timestamp_rfc_3339()
        for resource in resources:
            key = self.resource_status_key(resource)
            status = {
                "resource_id": resource["id"],
                "resource_type": resource["resource_type"],
                "workload_id": resource["workload_id"],
                "app_name": self.app["name"],
                "start": timestamp,
            }
            aws.upload_json_to_s3(status, key, self.bucket)

    def upload_resource_status_no_op(self, *resources):
        timestamp = util.now_timestamp_rfc_3339()
        for resource in resources:
            key = self.resource_status_key(resource)
            status = {
                "resource_id": resource["id"],
                "resource_type": resource["resource_type"],
                "workload_id": resource["workload_id"],
                "app_name": self.app["name"],
                "start": timestamp,
                "end": timestamp,
                "exit_code": "succeeded",
            }
            aws.upload_json_to_s3(status, key, self.bucket)

    def upload_resource_status_success(self, *resources):
        self.upload_resource_status_end("succeeded", *resources)

    def upload_resource_status_failed(self, *resources):
        self.upload_resource_status_end("failed", *resources)

    def upload_resource_status_end(self, exit_code, *resources):
        timestamp = util.now_timestamp_rfc_3339()
        for resource in resources:
            status = self.get_resource_status(resource)
            if status.get("end") != None:
                continue
            status["end"] = timestamp
            status["exit_code"] = exit_code
            key = self.resource_status_key(resource)
            aws.upload_json_to_s3(status, key, self.bucket)

    def resource_status_key(self, resource):
        return os.path.join(self.status_prefix, resource["id"], resource["workload_id"])


MODEL_IMPL_VALIDATION = {
    "required": [{"name": "create_estimator", "args": ["run_config", "model_config"]}],
    "optional": [
        {"name": "transform_tensorflow", "args": ["features", "labels", "model_config"]}
    ]
}

AGGREGATOR_IMPL_VALIDATION = {
    "required": [{"name": "aggregate_spark", "args": ["data", "columns", "args"]}]
}

TRANSFORMER_IMPL_VALIDATION = {
    "optional": [
        {"name": "transform_spark", "args": ["data", "columns", "args", "transformed_column"]},
        {"name": "reverse_transform_python", "args": ["transformed_value", "args"]},
        {"name": "transform_python", "args": ["sample", "args"]},
    ]
}


def _validate_impl(impl, impl_req):
    for optional_func in impl_req.get("optional", []):
        _validate_optional_fn_args(impl, optional_func["name"], optional_func["args"])

    for required_func in impl_req.get("required", []):
        _validate_required_fn_args(impl, required_func["name"], required_func["args"])


def _validate_required_fn(impl, fn_name):
    _validate_required_fn_args(impl, fn_name, None)


def _validate_optional_fn_args(impl, fn_name, args):
    if hasattr(impl, fn_name):
        _validate_required_fn_args(impl, fn_name, args)


def _validate_required_fn_args(impl, fn_name, args):
    fn = getattr(impl, fn_name, None)
    if not fn:
        raise UserException("function " + fn_name, "could not find function")

    if not callable(fn):
        raise UserException("function " + fn_name, "not a function")

    argspec = inspect.getargspec(fn)
    if argspec.varargs != None or argspec.keywords != None or argspec.defaults != None:
        raise UserException(
            "function " + fn_name,
            "invalid function signature, can only accept positional arguments",
        )

    if args:
        if argspec.args != args:
            raise UserException(
                "function " + fn_name,
                "expected function arguments arguments ({}) but found ({})".format(
                    ", ".join(args), ", ".join(argspec.args)
                ),
            )


def create_inputs_map(values_map, input_config):
    inputs = {}
    for input_name, input_config_item in input_config.items():
        if util.is_str(input_config_item):
            inputs[input_name] = values_map[input_config_item]
        elif util.is_int(input_config_item):
            inputs[input_name] = values_map[input_config_item]
        elif util.is_list(input_config_item):
            inputs[input_name] = [values_map[f] for f in input_config_item]
        else:
            raise CortexException("invalid column inputs")

    return inputs


def _deserialize_raw_ctx(raw_ctx):
    raw_columns = raw_ctx["raw_columns"]
    raw_ctx["raw_columns"] = util.merge_dicts_overwrite(
        raw_columns["raw_int_columns"],
        raw_columns["raw_float_columns"],
        raw_columns["raw_string_columns"],
    )

    data_split = raw_ctx["environment_data"]

    if data_split["csv_data"] is not None and data_split["parquet_data"] is None:
        raw_ctx["environment"]["data"] = data_split["csv_data"]
    elif data_split["parquet_data"] is not None and data_split["csv_data"] is None:
        raw_ctx["environment"]["data"] = data_split["parquet_data"]
    else:
        raise CortexException("expected csv_data or parquet_data but found " + data_split)
    return raw_ctx
