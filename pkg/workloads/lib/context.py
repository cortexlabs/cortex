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
from lib import util
from lib.storage import S3, LocalStorage
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
        elif "raw_obj" in kwargs:
            ctx_raw = kwargs["raw_obj"]
            self.ctx = _deserialize_raw_ctx(ctx_raw)
        elif "s3_path":
            local_ctx_path = os.path.join(self.cache_dir, "context.msgpack")
            bucket, key = S3.deconstruct_s3_path(kwargs["s3_path"])
            S3(bucket, client_config={}).download_file(key, local_ctx_path)
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
        self.estimators = self.ctx["estimators"]
        self.apis = self.ctx["apis"]
        self.training_datasets = {k: v["dataset"] for k, v in self.models.items()}
        self.api_version = self.cortex_config["api_version"]

        if "local_storage_path" in kwargs:
            self.storage = LocalStorage(base_dir=kwargs["local_storage_path"])
        else:
            self.storage = S3(
                bucket=self.cortex_config["bucket"],
                region=self.cortex_config["region"],
                client_config={},
            )

        if self.api_version != consts.CORTEX_VERSION:
            raise ValueError(
                "API version mismatch (Context: {}, Image: {})".format(
                    self.api_version, consts.CORTEX_VERSION
                )
            )

        if self.environment is not None:
            self.columns = util.merge_dicts_overwrite(self.raw_columns, self.transformed_columns)

            self.values = util.merge_dicts_overwrite(self.aggregates, self.constants)

            self.raw_column_names = list(self.raw_columns.keys())
            self.transformed_column_names = list(self.transformed_columns.keys())
            self.column_names = list(self.columns.keys())

        # Internal caches
        self._transformer_impls = {}
        self._aggregator_impls = {}
        self._estimator_impls = {}
        self._metadatas = {}
        self._obj_cache = {}
        self.spark_uploaded_impls = {}

        # This affects Tensorflow S3 access
        os.environ["AWS_REGION"] = self.cortex_config.get("region", "")

        # Id map
        self.id_map = {}
        self.pp_id_map = ResourceMap(self.python_packages) if self.python_packages else None
        self.rf_id_map = ResourceMap(self.raw_columns) if self.raw_columns else None
        self.ag_id_map = ResourceMap(self.aggregates) if self.aggregates else None
        self.tf_id_map = ResourceMap(self.transformed_columns) if self.transformed_columns else None
        self.td_id_map = ResourceMap(self.training_datasets) if self.training_datasets else None
        self.models_id_map = ResourceMap(self.models) if self.models else None
        self.constants_id_map = ResourceMap(self.constants) if self.constants else None
        self.id_map = util.merge_dicts_overwrite(
            self.pp_id_map,
            self.rf_id_map,
            self.ag_id_map,
            self.tf_id_map,
            self.td_id_map,
            self.models_id_map,
            self.constants_id_map,
        )

        self.apis_id_map = ResourceMap(self.apis)
        self.id_map = util.merge_dicts_overwrite(self.id_map, self.apis_id_map)

    def is_raw_column(self, name):
        return name in self.raw_columns

    def is_transformed_column(self, name):
        return name in self.transformed_columns

    def is_constant(self, name):
        return name in self.constants

    def is_aggregate(self, name):
        return name in self.aggregates

    def download_file(self, impl_key, cache_impl_path):
        if not os.path.isfile(cache_impl_path):
            self.storage.download_file(impl_key, cache_impl_path)
        return cache_impl_path

    def download_python_file(self, impl_key, module_name):
        cache_impl_path = os.path.join(self.cache_dir, "{}.py".format(module_name))
        self.download_file(impl_key, cache_impl_path)
        return cache_impl_path

    def get_obj(self, key):
        if key in self._obj_cache:
            return self._obj_cache[key]

        cache_path = os.path.join(self.cache_dir, key)
        self.download_file(key, cache_path)
        self._obj_cache[key] = util.read_msgpack(cache_path)
        return self._obj_cache[key]

    def load_module(self, module_prefix, module_name, impl_key):
        full_module_name = "{}_{}".format(module_prefix, module_name)

        try:
            impl_path = self.download_python_file(impl_key, full_module_name)
        except CortexException as e:
            e.wrap("unable to find python file " + module_name)
            raise

        try:
            impl = imp.load_source(full_module_name, impl_path)
        except Exception as e:
            raise UserException("unable to load python module " + module_name) from e

        return impl, impl_path

    def get_aggregator_impl(self, aggregate_name):
        aggregator_name = self.aggregates[aggregate_name]["aggregator"]
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
            e.wrap("aggregate " + aggregate_name, "aggregator")
            raise

        try:
            _validate_impl(impl, AGGREGATOR_IMPL_VALIDATION)
        except CortexException as e:
            e.wrap("aggregate " + aggregate_name, "aggregator " + aggregator["name"])
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

    def get_estimator_impl(self, model_name):
        estimator_name = self.models[model_name]["estimator"]
        if estimator_name in self._estimator_impls:
            return self._estimator_impls[estimator_name]

        estimator = self.estimators[estimator_name]

        module_prefix = "estimator"
        if "namespace" in estimator and estimator.get("namespace", None) is not None:
            module_prefix += "_" + estimator["namespace"]

        try:
            impl, impl_path = self.load_module(
                module_prefix, estimator["name"], estimator["impl_key"]
            )
        except CortexException as e:
            e.wrap("model " + model_name, "estimator")
            raise

        try:
            _validate_impl(impl, MODEL_IMPL_VALIDATION)
        except CortexException as e:
            e.wrap("model " + model_name, "estimator " + estimator["name"])
            raise

        self._estimator_impls[estimator_name] = (impl, impl_path)
        return (impl, impl_path)

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
        return self.storage.search(prefix=training_data_parts_prefix)

    def store_aggregate_result(self, result, aggregate):
        self.storage.put_msgpack(result, aggregate["key"])

    def extract_column_names(self, input):
        column_names = set()
        for resource_name in util.extract_resource_refs(input):
            if resource_name in self.columns:
                column_names.add(resource_name)
        return column_names

    def model_config(self, model_name):
        model = self.models[model_name]
        if model is None:
            return None
        estimator = self.estimators[model["estimator"]]
        target_column = self.columns[util.get_resource_ref(model["target_column"])]

        if estimator.get("target_column") is not None:
            target_col_type = self.get_inferred_column_type(target_column["name"])
            if target_col_type not in estimator["target_column"]:
                raise UserException(
                    "model " + model_name,
                    "target_column",
                    target_column["name"],
                    "unsupported type (expected type {}, got type {})".format(
                        util.data_type_str(estimator["target_column"]),
                        util.data_type_str(target_col_type),
                    ),
                )

        model_config = deepcopy(model)
        config_keys = [
            "name",
            "estimator"
            "estimator_path"
            "target_column"
            "input"
            "training_input"
            "hparams"
            "prediction_key"
            "data_partition_ratio"
            "training"
            "evaluation"
            "tags",
        ]
        util.keep_dict_keys(model_config, config_keys)

        model_config["target_column"] = target_column["name"]
        model_config["input"] = self.populate_values(
            model["input"], estimator["input"], preserve_column_refs=False
        )
        if model.get("training_input") is not None:
            model_config["training_input"] = self.populate_values(
                model["training_input"], estimator["training_input"], preserve_column_refs=False
            )
        if model.get("hparams") is not None:
            model_config["hparams"] = self.populate_values(
                model["hparams"], estimator["hparams"], preserve_column_refs=False
            )

        return model_config

    def get_resource_status(self, resource):
        key = self.resource_status_key(resource)
        return self.storage.get_json(key)

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
            self.storage.put_json(status, key)

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
            self.storage.put_json(status, key)

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
            self.storage.put_json(status, key)

    def resource_status_key(self, resource):
        return os.path.join(self.status_prefix, resource["id"], resource["workload_id"])

    def get_metadata_url(self, resource_id):
        return os.path.join(self.ctx["metadata_root"], resource_id + ".json")

    def write_metadata(self, resource_id, metadata):
        if resource_id in self._metadatas and self._metadatas[resource_id] == metadata:
            return

        self._metadatas[resource_id] = metadata
        self.storage.put_json(metadata, self.get_metadata_url(resource_id))

    def get_metadata(self, resource_id, use_cache=True):
        if use_cache and resource_id in self._metadatas:
            return self._metadatas[resource_id]

        metadata = self.storage.get_json(self.get_metadata_url(resource_id), allow_missing=True)
        self._metadatas[resource_id] = metadata
        return metadata

    def get_inferred_column_type(self, column_name):
        column = self.columns[column_name]
        column_type = self.columns[column_name]["type"]
        if column_type == consts.COLUMN_TYPE_INFERRED:
            column_type = self.get_metadata(column["id"])["type"]
            self.columns[column_name]["type"] = column_type

        return column_type

    # Replace aggregates and constants with their values, and columns with their names (unless preserve_column_refs == False)
    # Also validate against input_schema (if not None)
    def populate_values(self, input, input_schema, preserve_column_refs):
        if input is None:
            if input_schema is None:
                return None
            if input_schema.get("_allow_null") == True:
                return None
            raise UserException("Null value is not allowed")

        if util.is_resource_ref(input):
            res_name = util.get_resource_ref(input)
            if res_name in self.constants:
                const_val = self.constants[res_name]["value"]
                try:
                    return self.populate_values(const_val, input_schema, preserve_column_refs)
                except CortexException as e:
                    e.wrap("constant " + res_name)
                    raise

            if res_name in self.aggregates:
                agg_val = self.get_obj(self.aggregates[res_name]["key"])
                try:
                    return self.populate_values(agg_val, input_schema, preserve_column_refs)
                except CortexException as e:
                    e.wrap("aggregate " + res_name)
                    raise

            if res_name in self.columns:
                if input_schema is not None:
                    col_type = self.get_inferred_column_type(res_name)
                    if col_type not in input_schema["_type"]:
                        raise UserException(
                            "column {}: unsupported input type (expected type {}, got type {})".format(
                                res_name,
                                util.data_type_str(input_schema["_type"]),
                                util.data_type_str(col_type),
                            )
                        )
                if preserve_column_refs:
                    return input
                else:
                    return res_name

        if util.is_list(input):
            elem_schema = None
            if input_schema is not None:
                if not util.is_list(input_schema["_type"]):
                    raise UserException(
                        "unsupported input type (expected type {}, got {})".format(
                            util.data_type_str(input_schema["_type"]), util.user_obj_str(input)
                        )
                    )
                elem_schema = input_schema["_type"][0]

                min_count = input_schema.get("_min_count")
                if min_count is not None and len(input) < min_count:
                    raise UserException(
                        "list has length {}, but the minimum allowed length is {}".format(
                            len(input), min_count
                        )
                    )

                max_count = input_schema.get("_max_count")
                if max_count is not None and len(input) > max_count:
                    raise UserException(
                        "list has length {}, but the maximum allowed length is {}".format(
                            len(input), max_count
                        )
                    )

            casted = []
            for i, elem in enumerate(input):
                try:
                    casted.append(self.populate_values(elem, elem_schema, preserve_column_refs))
                except CortexException as e:
                    e.wrap("index " + i)
                    raise
            return casted

        if util.is_dict(input):
            if input_schema is None:
                casted = {}
                for key, val in input.items():
                    key_casted = self.populate_values(key, None, preserve_column_refs)
                    try:
                        val_casted = self.populate_values(val, None, preserve_column_refs)
                    except CortexException as e:
                        e.wrap(util.user_obj_str(key))
                        raise
                    casted[key_casted] = val_casted
                return casted

            if not util.is_dict(input_schema["_type"]):
                raise UserException(
                    "unsupported input type (expected type {}, got {})".format(
                        util.data_type_str(input_schema["_type"]), util.user_obj_str(input)
                    )
                )

            min_count = input_schema.get("_min_count")
            if min_count is not None and len(input) < min_count:
                raise UserException(
                    "map has length {}, but the minimum allowed length is {}".format(
                        len(input), min_count
                    )
                )

            max_count = input_schema.get("_max_count")
            if max_count is not None and len(input) > max_count:
                raise UserException(
                    "map has length {}, but the maximum allowed length is {}".format(
                        len(input), max_count
                    )
                )

            is_generic_map = False
            if len(input_schema["_type"]) == 1:
                input_type_key = next(iter(input_schema["_type"].keys()))
                if is_compound_type(input_type_key):
                    is_generic_map = True
                    generic_map_key_schema = input_schema_from_type_schema(input_type_key)
                    generic_map_value = input_schema["_type"][input_type_key]

            if is_generic_map:
                casted = {}
                for key, val in input.items():
                    key_casted = self.populate_values(
                        key, generic_map_key_schema, preserve_column_refs
                    )
                    try:
                        val_casted = self.populate_values(
                            val, generic_map_value, preserve_column_refs
                        )
                    except CortexException as e:
                        e.wrap(util.user_obj_str(key))
                        raise
                    casted[key_casted] = val_casted
                return casted

            # fixed map
            casted = {}
            for key, val_schema in input_schema["_type"].items():
                if key in input:
                    val = input[key]
                else:
                    if val_schema.get("_optional") is not True:
                        raise UserException("missing key: " + util.user_obj_str(key))
                    if val_schema.get("_default") is None:
                        continue
                    val = val_schema["_default"]

                try:
                    val_casted = self.populate_values(val, val_schema, preserve_column_refs)
                except CortexException as e:
                    e.wrap(util.user_obj_str(key))
                    raise
                casted[key] = val_casted
            return casted

        if input_schema is None:
            return input
        if not util.is_str(input_schema["_type"]):
            raise UserException(
                "unsupported input type (expected type {}, got {})".format(
                    util.data_type_str(input_schema["_type"]), util.user_obj_str(input)
                )
            )
        return cast_compound_type(input, input_schema["_type"])


def input_schema_from_type_schema(type_schema):
    return {
        "_type": type_schema,
        "_optional": False,
        "_default": None,
        "_allow_null": False,
        "_min_count": None,
        "_max_count": None,
    }


def is_compound_type(type_str):
    if not util.is_str(type_str):
        return False
    for subtype in type_str.split("|"):
        if subtype not in consts.ALL_TYPES:
            return False
    return True


def cast_compound_type(value, type_str):
    allowed_types = type_str.split("|")
    if consts.VALUE_TYPE_INT in allowed_types:
        if util.is_int(value):
            return value
    if consts.VALUE_TYPE_FLOAT in allowed_types:
        if util.is_int(value):
            return float(value)
        if util.is_float(value):
            return value
    if consts.VALUE_TYPE_STRING in allowed_types:
        if util.is_str(value):
            return value
    if consts.VALUE_TYPE_BOOL in allowed_types:
        if util.is_bool(value):
            return value

    raise UserException(
        "unsupported input type (expected type {}, got {})".format(
            util.data_type_str(type_str), util.user_obj_str(value)
        )
    )


MODEL_IMPL_VALIDATION = {
    "required": [{"name": "create_estimator", "args": ["run_config", "model_config"]}],
    "optional": [{"name": "transform_tensorflow", "args": ["features", "labels", "model_config"]}],
}

AGGREGATOR_IMPL_VALIDATION = {"required": [{"name": "aggregate_spark", "args": ["data", "input"]}]}

TRANSFORMER_IMPL_VALIDATION = {
    "optional": [
        {"name": "transform_spark", "args": ["data", "input", "transformed_column_name"]},
        # it is possible to not define transform_python() if you are only using the transformation for training columns, and not for inference
        {"name": "transform_python", "args": ["input"]},
        {"name": "reverse_transform_python", "args": ["transformed_value", "input"]},
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


def _deserialize_raw_ctx(raw_ctx):
    if raw_ctx["environment"]:
        raw_columns = raw_ctx["raw_columns"]
        raw_ctx["raw_columns"] = util.merge_dicts_overwrite(*raw_columns.values())

        data_split = raw_ctx["environment_data"]

        if data_split["csv_data"] is not None and data_split["parquet_data"] is None:
            raw_ctx["environment"]["data"] = data_split["csv_data"]
        elif data_split["parquet_data"] is not None and data_split["csv_data"] is None:
            raw_ctx["environment"]["data"] = data_split["parquet_data"]
        else:
            raise CortexException("expected csv_data or parquet_data but found " + data_split)

    return raw_ctx


# input should already have non-column arguments replaced, and all types validated
def create_transformer_inputs_from_map(input, col_value_map):
    if util.is_str(input):
        if util.is_resource_ref(input):
            res_name = util.get_resource_ref(input)
            return col_value_map[res_name]
        return input

    if util.is_list(input):
        replaced = []
        for item in input:
            replaced.append(create_transformer_inputs_from_map(item, col_value_map))
        return replaced

    if util.is_dict(input):
        replaced = {}
        for key, val in input.items():
            key_replaced = create_transformer_inputs_from_map(key, col_value_map)
            val_replaced = create_transformer_inputs_from_map(val, col_value_map)
            replaced[key_replaced] = val_replaced
        return replaced

    return input


# input should already have non-column arguments replaced, and all types validated
def create_transformer_inputs_from_lists(input, input_cols_sorted, col_values):
    col_value_map = {}
    for col_name, col_value in zip(input_cols_sorted, col_values):
        col_value_map[col_name] = col_value

    return create_transformer_inputs_from_map(input, col_value_map)
