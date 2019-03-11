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

from functools import reduce

import os

from pyspark.sql.types import *
from pyspark.sql.dataframe import DataFrame
import pyspark.sql.functions as F

from lib import util, aws
from lib.context import create_inputs_map
from lib.exceptions import CortexException, UserException, UserRuntimeException
from lib.log import get_logger
import consts

logger = get_logger()

CORTEX_TYPE_TO_SPARK_TYPE = {
    consts.COLUMN_TYPE_INT: LongType(),
    consts.COLUMN_TYPE_INT_LIST: ArrayType(LongType(), True),
    consts.COLUMN_TYPE_FLOAT: FloatType(),
    consts.COLUMN_TYPE_FLOAT_LIST: ArrayType(FloatType(), True),
    consts.COLUMN_TYPE_STRING: StringType(),
    consts.COLUMN_TYPE_STRING_LIST: ArrayType(StringType(), True),
}

FLOAT_PRECISION = 1e-4

CORTEX_TYPE_TO_ACCEPTABLE_SPARK_TYPES = {
    consts.COLUMN_TYPE_INT: [IntegerType(), LongType()],
    consts.COLUMN_TYPE_INT_LIST: [ArrayType(IntegerType(), True), ArrayType(LongType(), True)],
    consts.COLUMN_TYPE_FLOAT: [FloatType(), DoubleType()],
    consts.COLUMN_TYPE_FLOAT_LIST: [ArrayType(FloatType(), True), ArrayType(DoubleType(), True)],
    consts.COLUMN_TYPE_STRING: [StringType()],
    consts.COLUMN_TYPE_STRING_LIST: [ArrayType(StringType(), True)],
}


def accumulate_count(df, spark):
    acc = df._sc.accumulator(0)
    first_column_schema = df.schema[0]
    col_name = first_column_schema.name
    col_type = first_column_schema.dataType

    def _acc_func(val):
        acc.add(1)
        return val

    acc_func = F.udf(_acc_func, col_type)
    df = df.withColumn(col_name, acc_func(F.col(col_name)))
    return acc, df


def read_raw_dataset(ctx, spark):
    return spark.read.parquet(ctx.storage.hadoop_path(ctx.raw_dataset["key"]))


def log_df_schema(df, logger_func=logger.info):
    for line in df._jdf.schema().treeString().split("\n"):
        logger_func(line)


def write_training_data(model_name, df, ctx, spark):
    model = ctx.models[model_name]
    training_dataset = model["dataset"]
    column_names = model["feature_columns"] + [model["target_column"]] + model["training_columns"]

    df = df.select(*column_names)

    train_ratio = model["data_partition_ratio"]["training"]
    eval_ratio = model["data_partition_ratio"]["evaluation"]
    [train_df, eval_df] = df.randomSplit([train_ratio, eval_ratio])

    train_df_acc, train_df = accumulate_count(train_df, spark)
    train_df.write.mode("overwrite").format("tfrecords").option("recordType", "Example").save(
        ctx.storage.hadoop_path(training_dataset["train_key"])
    )

    eval_df_acc, eval_df = accumulate_count(eval_df, spark)
    eval_df.write.mode("overwrite").format("tfrecords").option("recordType", "Example").save(
        ctx.storage.hadoop_path(training_dataset["eval_key"])
    )

    metadata = {"training_size": train_df_acc.value, "eval_size": eval_df_acc.value}
    ctx.storage.put_json(metadata, training_dataset["metadata_key"])

    return df


def expected_schema_from_context(ctx):
    data_config = ctx.environment["data"]

    if data_config["type"] == "csv":
        expected_field_names = data_config["schema"]
    else:
        expected_field_names = [f["raw_column_name"] for f in data_config["schema"]]

    schema_fields = [
        StructField(
            name=fname,
            dataType=CORTEX_TYPE_TO_SPARK_TYPE[ctx.columns[fname]["type"]],
            nullable=not ctx.columns[fname].get("required", False),
        )
        for fname in expected_field_names
    ]
    return StructType(schema_fields)


def compare_column_schemas(expected_schema, actual_schema):
    # Nullables are being left out because when Spark is reading CSV files, it is setting nullable to true
    # regardless of if the column has all values or not. The null checks will be done elsewhere.
    # This compares only the schemas
    expected_sorted_fields = sorted(
        [(f.name, f.dataType) for f in expected_schema], key=lambda f: f[0]
    )

    # Sorted for determinism when testing
    actual_sorted_fields = sorted([(f.name, f.dataType) for f in actual_schema], key=lambda f: f[0])

    return expected_sorted_fields == actual_sorted_fields


def min_check(input_col, min):
    return input_col >= min, input_col < min


def max_check(input_col, max):
    return input_col <= max, input_col > max


def required_check(input_col, required):
    if required is True:
        return input_col.isNotNull(), input_col.isNull()
    else:
        return None


def values_check(input_col, values):
    return input_col.isin(values), input_col.isin(values) == False


def generate_conditions(condition_map, raw_column_config, input_col):
    for cond_name in condition_map.keys():
        if raw_column_config.get(cond_name) is not None:
            cond_col_tuple = condition_map[cond_name](input_col, raw_column_config[cond_name])
            if cond_col_tuple is not None:
                positive_cond_col, cond_col = cond_col_tuple
                yield {
                    "col_name": raw_column_config["name"],
                    "cond_name": cond_name,
                    "cond_col": cond_col,
                    "positive_cond_str": str(positive_cond_col._jc),
                }


def value_checker(raw_column_config):
    condition_map = {
        "min": min_check,
        "max": max_check,
        "required": required_check,
        "values": values_check,
    }

    input_col = F.col(raw_column_config["name"])

    return generate_conditions(condition_map, raw_column_config, input_col)


def value_check_data(ctx, df, raw_columns=None):
    # merge the list of list of conditions
    if raw_columns is None:
        raw_columns = list(ctx.rf_id_map.keys())
    column_value_checkers_list = [
        cvc for f in raw_columns for cvc in value_checker(ctx.rf_id_map[f])
    ]

    cvc_dict = {
        ("{}_{}".format(d["col_name"], d["cond_name"])): d for d in column_value_checkers_list
    }

    # map each value for each column condition to 1 if it is satisfied otherwise 0 and sum them
    # conditions automatically skip nulls (unless you are checking for nulls)
    aggregate_column_value_checkers = [
        F.sum(F.when(d["cond_col"], 1).otherwise(0)).alias(cond_col_name)
        for cond_col_name, d in cvc_dict.items()
    ]

    if len(aggregate_column_value_checkers) == 0:
        return {}

    results_dict = df.agg(*aggregate_column_value_checkers).collect()[0].asDict()

    condition_stats = [(cond_col_name, count) for cond_col_name, count in results_dict.items()]

    conditions_dict = {}
    # group each the violations by column
    for cond_col_name, count in condition_stats:
        if count != 0:
            col_name = cvc_dict[cond_col_name]["col_name"]
            positive_cond_str = cvc_dict[cond_col_name]["positive_cond_str"]
            check_list = conditions_dict.get(col_name, [])

            check_list.append((positive_cond_str, count))
            conditions_dict[col_name] = check_list

    return conditions_dict


def ingest(ctx, spark):
    expected_schema = expected_schema_from_context(ctx)

    if ctx.environment["data"]["type"] == "csv":
        df = read_csv(ctx, spark)
    elif ctx.environment["data"]["type"] == "parquet":
        df = read_parquet(ctx, spark)

    if compare_column_schemas(expected_schema, df.schema) is not True:
        logger.error("expected schema:")
        log_df_schema(spark.createDataFrame([], expected_schema), logger.error)
        logger.error("found schema:")
        log_df_schema(df, logger.error)

        raise UserException("raw data schema mismatch")

    return df


def read_csv(ctx, spark):
    data_config = ctx.environment["data"]
    schema = expected_schema_from_context(ctx)

    csv_config = {
        util.snake_to_camel(param_name): val
        for param_name, val in data_config.get("csv_config", {}).items()
        if val is not None
    }

    return spark.read.csv(data_config["path"], schema=schema, mode="FAILFAST", **csv_config)


def read_parquet(ctx, spark):
    parquet_config = ctx.environment["data"]
    df = spark.read.parquet(parquet_config["path"])

    parquet_columns = [c["parquet_column_name"] for c in parquet_config["schema"]]
    missing_cols = util.subtract_lists(parquet_columns, df.columns)
    if len(missing_cols) > 0:
        raise UserException("parquet dataset", "missing columns: " + str(missing_cols))

    selectExprs = [
        "{} as {}".format(c["parquet_column_name"], c["raw_column_name"])
        for c in parquet_config["schema"]
    ]

    return df.selectExpr(*selectExprs)


def column_names_to_index(columns_input_config):
    column_list = []
    for k, v in columns_input_config.items():
        if util.is_list(v):
            column_list += v
        else:
            column_list.append(v)

    required_input_columns_sorted = sorted(set(column_list))

    index_to_col_map = dict(
        [(column_name, idx) for idx, column_name in enumerate(required_input_columns_sorted)]
    )

    columns_input_config_indexed = create_inputs_map(index_to_col_map, columns_input_config)
    return required_input_columns_sorted, columns_input_config_indexed


# not included in this list: collect_list, grouping, grouping_id
AGG_SPARK_LIST = set(
    [
        "approx_count_distinct",
        "avg",
        "collect_set",
        "count",
        "countDistinct",
        "kurtosis",
        "max",
        "mean",
        "min",
        "skewness",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "sumDistinct",
        "var_pop",
        "var_samp",
        "variance",
    ]
)


def extract_spark_name(f_name):
    if f_name.endswith("_string") or f_name.endswith("_float") or f_name.endswith("_int"):
        f_name = "_".join(f_name.split("_")[:-1])
    snake_case_mapping = {"sum_distinct": "sumDistinct", "count_distinct": "countDistinct"}
    return snake_case_mapping.get(f_name, f_name)


def split_aggregators(columns_to_aggregate, ctx):
    aggregate_resources = [ctx.aggregates[r] for r in columns_to_aggregate]

    builtin_aggregates = []
    custom_aggregates = []

    for r in aggregate_resources:
        aggregator = ctx.aggregators[r["aggregator"]]
        spark_name = extract_spark_name(aggregator["name"])
        if aggregator.get("namespace", None) == "cortex" and spark_name in AGG_SPARK_LIST:
            builtin_aggregates.append(r)
        else:
            custom_aggregates.append(r)

    return builtin_aggregates, custom_aggregates


def run_builtin_aggregators(builtin_aggregates, df, ctx, spark):
    agg_cols = []
    for r in builtin_aggregates:
        aggregator = ctx.aggregators[r["aggregator"]]
        f_name = extract_spark_name(aggregator["name"])

        agg_func = getattr(F, f_name)
        col_name_list = []
        columns_dict = r["inputs"]["columns"]

        if "col" in columns_dict.keys():
            col_name_list.append(columns_dict["col"])
        if "cols" in columns_dict.keys():
            col_name_list += columns_dict["cols"]
        if "col1" in columns_dict.keys() and "col2" in columns_dict.keys():
            col_name_list.append(columns_dict["col1"])
            col_name_list.append(columns_dict["col2"])

        if len(col_name_list) == 0:
            raise CortexException("input columns not found in aggregator: {}".format(r))

        args = {}
        if r["inputs"].get("args", None) is not None and len(r["inputs"]["args"]) > 0:
            args = ctx.populate_args(r["inputs"]["args"])
        col_list = [F.col(c) for c in col_name_list]
        agg_cols.append(agg_func(*col_list, **args).alias(r["name"]))

    results = df.agg(*agg_cols).collect()[0].asDict()

    for r in builtin_aggregates:
        ctx.store_aggregate_result(results[r["name"]], r)

    return results


def run_custom_aggregator(aggregator_resource, df, ctx, spark):
    aggregator = ctx.aggregators[aggregator_resource["aggregator"]]
    aggregate_name = aggregator_resource["name"]
    aggregator_impl, _ = ctx.get_aggregator_impl(aggregate_name)
    input_schema = aggregator_resource["inputs"]
    aggregator_column_input = input_schema["columns"]
    args_schema = input_schema["args"]
    args = {}
    if input_schema.get("args", None) is not None and len(input_schema["args"]) > 0:
        args = ctx.populate_args(input_schema["args"])
    try:
        result = aggregator_impl.aggregate_spark(df, aggregator_column_input, args)
    except Exception as e:
        raise UserRuntimeException(
            "aggregate " + aggregator_resource["name"],
            "aggregator " + aggregator["name"],
            "function aggregate_spark",
        ) from e

    if not util.validate_value_type(result, aggregator["output_type"]):
        raise UserException(
            "aggregate " + aggregator_resource["name"],
            "aggregator " + aggregator["name"],
            "type of {} is not {}".format(
                util.str_rep(util.pp_str(result), truncate=100), aggregator["output_type"]
            ),
        )

    ctx.store_aggregate_result(result, aggregator_resource)
    return result


def extract_inputs(column_name, ctx):
    columns_input_config = ctx.transformed_columns[column_name]["inputs"]["columns"]
    impl_args_schema = ctx.transformed_columns[column_name]["inputs"]["args"]
    if impl_args_schema is not None:
        impl_args = ctx.populate_args(impl_args_schema)
    else:
        impl_args = {}
    return columns_input_config, impl_args


def execute_transform_spark(column_name, df, ctx, spark):
    trans_impl, trans_impl_path = ctx.get_transformer_impl(column_name)
    spark.sparkContext.addPyFile(trans_impl_path)  # Executor pods need this because of the UDF
    columns_input_config, impl_args = extract_inputs(column_name, ctx)
    try:
        return trans_impl.transform_spark(df, columns_input_config, impl_args, column_name)
    except Exception as e:
        raise UserRuntimeException("function transform_spark") from e


def execute_transform_python(column_name, df, ctx, spark, validate=False):
    trans_impl, trans_impl_path = ctx.get_transformer_impl(column_name)
    columns_input_config, impl_args = extract_inputs(column_name, ctx)

    spark.sparkContext.addPyFile(trans_impl_path)  # Executor pods need this because of the UDF
    # not a dictionary because it is possible that one column may map to multiple input names
    required_columns_sorted, columns_input_config_indexed = column_names_to_index(
        columns_input_config
    )

    def _transform(*values):
        inputs = create_inputs_map(values, columns_input_config_indexed)
        return trans_impl.transform_python(inputs, impl_args)

    transform_python_func = _transform

    if validate:
        transformed_column = ctx.transformed_columns[column_name]

        def _transform_and_validate(*values):
            result = _transform(*values)
            if not util.validate_column_type(result, transformed_column["type"]):
                raise UserException(
                    "transformed column " + column_name,
                    "tranformation " + transformed_column["transformer"],
                    "type of {} is not {}".format(result, transformed_column["type"]),
                )

            return result

        transform_python_func = _transform_and_validate

    column_data_type_str = ctx.transformed_columns[column_name]["type"]
    transform_udf = F.udf(transform_python_func, CORTEX_TYPE_TO_SPARK_TYPE[column_data_type_str])
    return df.withColumn(column_name, transform_udf(*required_columns_sorted))


def validate_transformer(column_name, df, ctx, spark):
    transformed_column = ctx.transformed_columns[column_name]

    trans_impl, _ = ctx.get_transformer_impl(column_name)

    if hasattr(trans_impl, "transform_python"):
        try:
            transform_python_collect = execute_transform_python(
                column_name, df, ctx, spark, validate=True
            ).collect()
        except Exception as e:
            logger.exception("hello")
            raise UserRuntimeException(
                "transformed column " + column_name,
                transformed_column["transformer"] + ".transform_python",
            ) from e

    if hasattr(trans_impl, "transform_spark"):

        try:
            transform_spark_df = execute_transform_spark(column_name, df, ctx, spark)

            # check that the return object is a dataframe
            if type(transform_spark_df) is not DataFrame:
                raise UserException(
                    "expected pyspark.sql.dataframe.DataFrame but found type {}".format(
                        type(transform_spark_df)
                    )
                )

            # check that a column is added with the expected name
            if column_name not in transform_spark_df.columns:
                logger.error("schema of output dataframe:")
                log_df_schema(transform_spark_df, logger.error)

                raise UserException(
                    "output dataframe after running transformer does not have column {}".format(
                        column_name
                    )
                )

            # check that transformer run on data
            try:
                transform_spark_df.select(column_name).collect()
            except Exception as e:
                raise UserRuntimeException("function transform_spark") from e

            actual_structfield = transform_spark_df.select(column_name).schema.fields[0]

            # check that expected output column has the correct data type
            if (
                actual_structfield.dataType
                not in CORTEX_TYPE_TO_ACCEPTABLE_SPARK_TYPES[transformed_column["type"]]
            ):
                raise UserException(
                    "incorrect column type, expected {}, found {}.".format(
                        " or ".join(
                            str(t)
                            for t in CORTEX_TYPE_TO_ACCEPTABLE_SPARK_TYPES[
                                transformed_column["type"]
                            ]
                        ),
                        actual_structfield.dataType,
                    )
                )

            # perform the necessary upcast/downcast for the column e.g INT -> LONG or DOUBLE -> FLOAT
            transform_spark_df = transform_spark_df.withColumn(
                column_name,
                F.col(column_name).cast(
                    CORTEX_TYPE_TO_SPARK_TYPE[ctx.transformed_columns[column_name]["type"]]
                ),
            )

            # check that the function doesn't modify the schema of the other columns in the input dataframe
            if set(transform_spark_df.columns) - set([column_name]) != set(df.columns):
                logger.error("expected schema:")

                log_df_schema(df, logger.error)

                logger.error("found schema (with {} dropped):".format(column_name))
                log_df_schema(transform_spark_df.drop(column_name), logger.error)

                raise UserException(
                    "a column besides {} was modifed in the output dataframe".format(column_name)
                )
        except CortexException as e:
            e.wrap(
                "transformed column " + column_name,
                transformed_column["transformer"] + ".transform_spark",
            )
            raise

        if hasattr(trans_impl, "transform_spark") and hasattr(trans_impl, "transform_python"):
            name_type_map = [(s.name, s.dataType) for s in transform_spark_df.schema]
            transform_spark_collect = transform_spark_df.collect()

            for tp_row, ts_row in zip(transform_python_collect, transform_spark_collect):
                tp_dict = tp_row.asDict()
                ts_dict = ts_row.asDict()

                for name, dataType in name_type_map:
                    if tp_dict[name] == ts_dict[name]:
                        continue
                    elif dataType == FloatType() and util.isclose(
                        tp_dict[name], ts_dict[name], FLOAT_PRECISION
                    ):
                        continue
                    raise UserException(
                        column_name,
                        "{0}.transform_spark and {0}.transform_python had differing values".format(
                            transformed_column["transformer"]
                        ),
                        "{} != {}".format(ts_row, tp_row),
                    )


def transform_column(column_name, df, ctx, spark):
    if not ctx.is_transformed_column(column_name):
        return df
    if column_name in df.columns:
        return df
    transformed_column = ctx.transformed_columns[column_name]

    trans_impl, trans_impl_path = ctx.get_transformer_impl(column_name)
    if hasattr(trans_impl, "transform_spark"):
        return execute_transform_spark(column_name, df, ctx, spark).withColumn(
            column_name,
            F.col(column_name).cast(
                CORTEX_TYPE_TO_SPARK_TYPE[ctx.transformed_columns[column_name]["type"]]
            ),
        )
    elif hasattr(trans_impl, "transform_python"):
        return execute_transform_python(column_name, df, ctx, spark)
    else:
        raise UserException(
            "transformed column " + column_name,
            "transformer " + transformed_column["transformer"],
            "transform_spark(), transform_python(), or both must be defined",
        )


def transform(model_name, accumulated_df, ctx, spark):
    model = ctx.models[model_name]
    column_names = model["feature_columns"] + [model["target_column"]] + model["training_columns"]

    for column_name in column_names:
        accumulated_df = transform_column(column_name, accumulated_df, ctx, spark)

    return accumulated_df
