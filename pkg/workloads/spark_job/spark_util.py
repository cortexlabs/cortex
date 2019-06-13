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

from lib import util
from lib.context import create_transformer_inputs_from_map, create_transformer_inputs_from_lists
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
    consts.COLUMN_TYPE_FLOAT: [FloatType(), DoubleType(), IntegerType(), LongType()],
    consts.COLUMN_TYPE_FLOAT_LIST: [
        ArrayType(FloatType(), True),
        ArrayType(DoubleType(), True),
        ArrayType(IntegerType(), True),
        ArrayType(LongType(), True),
    ],
    consts.COLUMN_TYPE_STRING: [StringType()],
    consts.COLUMN_TYPE_STRING_LIST: [ArrayType(StringType(), True)],
}

CORTEX_TYPE_TO_ACCEPTABLE_PYTHON_TYPE_STRS = {
    consts.COLUMN_TYPE_INT: ["int"],
    consts.COLUMN_TYPE_INT_LIST: ["[int]"],
    consts.COLUMN_TYPE_FLOAT: ["float", "int"],
    consts.COLUMN_TYPE_FLOAT_LIST: ["[float]", "[int]"],
    consts.COLUMN_TYPE_STRING: ["string"],
    consts.COLUMN_TYPE_STRING_LIST: ["[string]"],
}

CORTEX_TYPE_TO_CASTABLE_SPARK_TYPES = {
    "csv": {
        consts.COLUMN_TYPE_INT: [IntegerType(), LongType()],
        consts.COLUMN_TYPE_INT_LIST: [ArrayType(IntegerType(), True), ArrayType(LongType(), True)],
        consts.COLUMN_TYPE_FLOAT: [FloatType(), DoubleType(), IntegerType(), LongType()],
        consts.COLUMN_TYPE_FLOAT_LIST: [
            ArrayType(FloatType(), True),
            ArrayType(DoubleType(), True),
        ],
        consts.COLUMN_TYPE_STRING: [
            StringType(),
            IntegerType(),
            LongType(),
            FloatType(),
            DoubleType(),
            ArrayType(FloatType(), True),
            ArrayType(DoubleType(), True),
            ArrayType(StringType(), True),
            ArrayType(IntegerType(), True),
            ArrayType(LongType(), True),
        ],
        consts.COLUMN_TYPE_STRING_LIST: [ArrayType(StringType(), True)],
    },
    "parquet": {
        consts.COLUMN_TYPE_INT: [IntegerType(), LongType()],
        consts.COLUMN_TYPE_INT_LIST: [ArrayType(IntegerType(), True), ArrayType(LongType(), True)],
        consts.COLUMN_TYPE_FLOAT: [FloatType(), DoubleType()],
        consts.COLUMN_TYPE_FLOAT_LIST: [
            ArrayType(FloatType(), True),
            ArrayType(DoubleType(), True),
        ],
        consts.COLUMN_TYPE_STRING: [StringType()],
        consts.COLUMN_TYPE_STRING_LIST: [
            ArrayType(StringType(), True),
            ArrayType(FloatType(), True),
            ArrayType(DoubleType(), True),
            ArrayType(IntegerType(), True),
            ArrayType(LongType(), True),
        ],
    },
}

PYTHON_TYPE_TO_CORTEX_TYPE = {
    int: consts.COLUMN_TYPE_INT,
    float: consts.COLUMN_TYPE_FLOAT,
    str: consts.COLUMN_TYPE_STRING,
}

PYTHON_TYPE_TO_CORTEX_LIST_TYPE = {
    int: consts.COLUMN_TYPE_INT_LIST,
    float: consts.COLUMN_TYPE_FLOAT_LIST,
    str: consts.COLUMN_TYPE_STRING_LIST,
}

SPARK_TYPE_TO_CORTEX_TYPE = {
    IntegerType(): consts.COLUMN_TYPE_INT,
    LongType(): consts.COLUMN_TYPE_INT,
    ArrayType(IntegerType(), True): consts.COLUMN_TYPE_INT_LIST,
    ArrayType(LongType(), True): consts.COLUMN_TYPE_INT_LIST,
    FloatType(): consts.COLUMN_TYPE_FLOAT,
    DoubleType(): consts.COLUMN_TYPE_FLOAT,
    ArrayType(FloatType(), True): consts.COLUMN_TYPE_FLOAT_LIST,
    ArrayType(DoubleType(), True): consts.COLUMN_TYPE_FLOAT_LIST,
    StringType(): consts.COLUMN_TYPE_STRING,
    ArrayType(StringType(), True): consts.COLUMN_TYPE_STRING_LIST,
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
    column_names = ctx.extract_column_names(
        [model["input"], model["target_column"], model.get("training_input")]
    )

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

    ctx.write_metadata(
        training_dataset["id"],
        {"training_size": train_df_acc.value, "eval_size": eval_df_acc.value},
    )

    return df


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
    fileType = ctx.environment["data"]["type"]
    if fileType == "csv":
        df = read_csv(ctx, spark)
    elif fileType == "parquet":
        df = read_parquet(ctx, spark)

    input_type_map = {f.name: f.dataType for f in df.schema}
    for raw_column_name in ctx.raw_columns:
        raw_column = ctx.raw_columns[raw_column_name]
        expected_cortex_type = raw_column["type"]
        actual_spark_type = input_type_map[raw_column_name]

        if expected_cortex_type == consts.COLUMN_TYPE_INFERRED:
            if actual_spark_type not in SPARK_TYPE_TO_CORTEX_TYPE:
                df = df.withColumn(raw_column_name, F.col(raw_column_name).cast(StringType()))
            else:
                actual_cortex_type = SPARK_TYPE_TO_CORTEX_TYPE[actual_spark_type]
                expected_spark_type = CORTEX_TYPE_TO_SPARK_TYPE[actual_cortex_type]
                if actual_spark_type != expected_spark_type:
                    df = df.withColumn(
                        raw_column_name, F.col(raw_column_name).cast(expected_spark_type)
                    )
        else:
            expected_spark_type = CORTEX_TYPE_TO_SPARK_TYPE[expected_cortex_type]
            if actual_spark_type in SPARK_TYPE_TO_CORTEX_TYPE:
                expected_types = CORTEX_TYPE_TO_CASTABLE_SPARK_TYPES[fileType][expected_cortex_type]
                if actual_spark_type not in expected_types:
                    logger.error("found schema:")
                    log_df_schema(df, logger.error)

                    raise UserException(
                        "raw column " + raw_column_name,
                        "type mismatch",
                        "expected {} but got {}".format(
                            " or ".join(str(x) for x in expected_types), actual_spark_type
                        ),
                    )
                if actual_spark_type != expected_spark_type:
                    df = df.withColumn(
                        raw_column_name, F.col(raw_column_name).cast(expected_spark_type)
                    )
            else:
                try:
                    df = df.withColumn(
                        raw_column_name, F.col(raw_column_name).cast(expected_spark_type)
                    )
                except Exception as e:
                    raise UserException(
                        "tried casting " + raw_column_name,
                        "from ingested type " + actual_spark_type,
                        "to expected type " + expected_spark_type,
                        "but got exception: " + e,
                    )

    return df.select(*sorted(df.columns))


def read_csv(ctx, spark):
    data_config = ctx.environment["data"]

    csv_config = {
        util.snake_to_camel(param_name): val
        for param_name, val in data_config.get("csv_config", {}).items()
        if val is not None
    }

    df = spark.read.csv(data_config["path"], inferSchema=True, mode="FAILFAST", **csv_config)
    if len(data_config["schema"]) != len(df.columns):
        raise UserException(
            "expected " + len(data_config["schema"]) + " column(s) but got " + len(df.columns)
        )

    col_names = [util.get_resource_ref(col_ref) for col_ref in data_config["schema"]]
    renamed_cols = [F.col(c).alias(col_names[idx]) for idx, c in enumerate(df.columns)]
    return df.select(*renamed_cols)


def read_parquet(ctx, spark):
    parquet_config = ctx.environment["data"]
    df = spark.read.parquet(parquet_config["path"])

    alias_map = {}
    for parquet_col_config in parquet_config["schema"]:
        col_name = util.get_resource_ref(parquet_col_config["raw_column"])
        if col_name in ctx.raw_columns:
            alias_map[col_name] = parquet_col_config["parquet_column_name"]

    missing_cols = set(alias_map.keys()) - set(df.columns)
    if len(missing_cols) > 0:
        logger.error("found schema:")
        log_df_schema(df, logger.error)
        raise UserException("missing column(s) in input dataset", str(missing_cols))

    selectExprs = [
        "{} as {}".format(parq_name, col_name) for col_name, parq_name in alias_map.items()
    ]

    return df.selectExpr(*selectExprs)


# not included in this list: collect_list, grouping, grouping_id
AGG_SPARK_LIST = {
    "approx_count_distinct",
    "avg",
    "collect_set_int",
    "collect_set_float",
    "collect_set_string",
    "count",
    "count_distinct",
    "covar_pop",
    "covar_samp",
    "kurtosis",
    "max_int",
    "max_float",
    "max_string",
    "mean",
    "min_int",
    "min_float",
    "min_string",
    "skewness",
    "stddev",
    "stddev_pop",
    "stddev_samp",
    "sum_int",
    "sum_float",
    "sum_distinct_int",
    "sum_distinct_float",
    "var_pop",
    "var_samp",
    "variance",
}


def split_aggregators(aggregate_names, ctx):
    aggregates = [ctx.aggregates[agg_name] for agg_name in aggregate_names]

    builtin_aggregates = []
    custom_aggregates = []

    for agg in aggregates:
        aggregator = ctx.aggregators[agg["aggregator"]]
        if aggregator.get("namespace", None) == "cortex" and aggregator["name"] in AGG_SPARK_LIST:
            builtin_aggregates.append(agg)
        else:
            custom_aggregates.append(agg)

    return builtin_aggregates, custom_aggregates


def run_builtin_aggregators(builtin_aggregates, df, ctx, spark):
    agg_cols = [get_builtin_aggregator_column(agg, ctx) for agg in builtin_aggregates]
    results = df.agg(*agg_cols).collect()[0].asDict()

    for agg in builtin_aggregates:
        result = results[agg["name"]]
        aggregator = ctx.aggregators[agg["aggregator"]]
        result = util.cast_output_type(result, aggregator["output_type"])

        results[agg["name"]] = result
        ctx.store_aggregate_result(result, agg)

    return results


def get_builtin_aggregator_column(agg, ctx):
    try:
        aggregator = ctx.aggregators[agg["aggregator"]]

        try:
            input = ctx.populate_values(
                agg["input"], aggregator["input"], preserve_column_refs=False
            )
        except CortexException as e:
            e.wrap("input")
            raise

        if aggregator["name"] == "approx_count_distinct":
            return F.approxCountDistinct(input["col"], input.get("rsd")).alias(agg["name"])
        if aggregator["name"] == "avg":
            return F.avg(input).alias(agg["name"])
        if aggregator["name"] in {"collect_set_int", "collect_set_float", "collect_set_string"}:
            return F.collect_set(input).alias(agg["name"])
        if aggregator["name"] == "count":
            return F.count(input).alias(agg["name"])
        if aggregator["name"] == "count_distinct":
            return F.countDistinct(*input).alias(agg["name"])
        if aggregator["name"] == "covar_pop":
            return F.covar_pop(input["col1"], input["col2"]).alias(agg["name"])
        if aggregator["name"] == "covar_samp":
            return F.covar_samp(input["col1"], input["col2"]).alias(agg["name"])
        if aggregator["name"] == "kurtosis":
            return F.kurtosis(input).alias(agg["name"])
        if aggregator["name"] in {"max_int", "max_float", "max_string"}:
            return F.max(input).alias(agg["name"])
        if aggregator["name"] == "mean":
            return F.mean(input).alias(agg["name"])
        if aggregator["name"] in {"min_int", "min_float", "min_string"}:
            return F.min(input).alias(agg["name"])
        if aggregator["name"] == "skewness":
            return F.skewness(input).alias(agg["name"])
        if aggregator["name"] == "stddev":
            return F.stddev(input).alias(agg["name"])
        if aggregator["name"] == "stddev_pop":
            return F.stddev_pop(input).alias(agg["name"])
        if aggregator["name"] == "stddev_samp":
            return F.stddev_samp(input).alias(agg["name"])
        if aggregator["name"] in {"sum_int", "sum_float"}:
            return F.sum(input).alias(agg["name"])
        if aggregator["name"] in {"sum_distinct_int", "sum_distinct_float"}:
            return F.sumDistinct(input).alias(agg["name"])
        if aggregator["name"] == "var_pop":
            return F.var_pop(input).alias(agg["name"])
        if aggregator["name"] == "var_samp":
            return F.var_samp(input).alias(agg["name"])
        if aggregator["name"] == "variance":
            return F.variance(input).alias(agg["name"])

        raise ValueError("missing builtin aggregator")  # unexpected

    except CortexException as e:
        e.wrap("aggregate " + agg["name"])
        raise


def run_custom_aggregator(aggregate, df, ctx, spark):
    aggregator = ctx.aggregators[aggregate["aggregator"]]
    aggregator_impl, _ = ctx.get_aggregator_impl(aggregate["name"])

    try:
        input = ctx.populate_values(
            aggregate["input"], aggregator["input"], preserve_column_refs=False
        )
    except CortexException as e:
        e.wrap("aggregate " + aggregate["name"], "input")
        raise

    try:
        result = aggregator_impl.aggregate_spark(df, input)
    except Exception as e:
        raise UserRuntimeException(
            "aggregate " + aggregate["name"],
            "aggregator " + aggregator["name"],
            "function aggregate_spark",
        ) from e

    if aggregator.get("output_type") is not None and not util.validate_output_type(
        result, aggregator["output_type"]
    ):
        raise UserException(
            "aggregate " + aggregate["name"],
            "aggregator " + aggregator["name"],
            "unsupported return type (expected type {}, got {})".format(
                util.data_type_str(aggregator["output_type"]), util.user_obj_str(result)
            ),
        )

    result = util.cast_output_type(result, aggregator["output_type"])
    ctx.store_aggregate_result(result, aggregate)
    return result


def execute_transform_spark(column_name, df, ctx, spark):
    trans_impl, trans_impl_path = ctx.get_transformer_impl(column_name)
    transformed_column = ctx.transformed_columns[column_name]
    transformer = ctx.transformers[transformed_column["transformer"]]

    if trans_impl_path not in ctx.spark_uploaded_impls:
        spark.sparkContext.addPyFile(trans_impl_path)  # Executor pods need this because of the UDF
        ctx.spark_uploaded_impls[trans_impl_path] = True

    try:
        input = ctx.populate_values(
            transformed_column["input"], transformer["input"], preserve_column_refs=False
        )
    except CortexException as e:
        e.wrap("input")
        raise

    try:
        return trans_impl.transform_spark(df, input, column_name)
    except Exception as e:
        raise UserRuntimeException("function transform_spark") from e


def execute_transform_python(column_name, df, ctx, spark, validate=False):
    trans_impl, trans_impl_path = ctx.get_transformer_impl(column_name)
    transformed_column = ctx.transformed_columns[column_name]
    transformer = ctx.transformers[transformed_column["transformer"]]

    input_cols_sorted = sorted(ctx.extract_column_names(transformed_column["input"]))

    try:
        input = ctx.populate_values(
            transformed_column["input"], transformer["input"], preserve_column_refs=True
        )
    except CortexException as e:
        e.wrap("input")
        raise

    if trans_impl_path not in ctx.spark_uploaded_impls:
        spark.sparkContext.addPyFile(trans_impl_path)  # Executor pods need this because of the UDF
        ctx.spark_uploaded_impls[trans_impl_path] = True

    def _transform(*values):
        transformer_input = create_transformer_inputs_from_lists(input, input_cols_sorted, values)
        return trans_impl.transform_python(transformer_input)

    transform_python_func = _transform

    if validate:
        transformed_column = ctx.transformed_columns[column_name]
        column_type = ctx.get_inferred_column_type(column_name)

        def _transform_and_validate(*values):
            result = _transform(*values)
            if not util.validate_cortex_type(result, column_type):
                raise UserException(
                    "transformed column " + column_name,
                    "tranformer " + transformed_column["transformer"],
                    "incorrect return value type: expected {}, got {}.".format(
                        " or ".join(CORTEX_TYPE_TO_ACCEPTABLE_PYTHON_TYPE_STRS[column_type]),
                        util.user_obj_str(result),
                    ),
                )

            return result

        transform_python_func = _transform_and_validate

    column_type = ctx.get_inferred_column_type(column_name)
    transform_udf = F.udf(transform_python_func, CORTEX_TYPE_TO_SPARK_TYPE[column_type])
    return df.withColumn(column_name, transform_udf(*input_cols_sorted))


def infer_python_type(obj):
    obj_type = type(obj)

    if obj_type == list:
        obj_type = type(obj[0])
        return PYTHON_TYPE_TO_CORTEX_LIST_TYPE[obj_type]

    return PYTHON_TYPE_TO_CORTEX_TYPE[obj_type]


def validate_transformer(column_name, test_df, ctx, spark):
    transformed_column = ctx.transformed_columns[column_name]
    transformer = ctx.transformers[transformed_column["transformer"]]
    trans_impl, _ = ctx.get_transformer_impl(column_name)

    inferred_python_type = None
    inferred_spark_type = None

    if hasattr(trans_impl, "transform_python"):
        try:
            if transformer["output_type"] == consts.COLUMN_TYPE_INFERRED:
                sample_df = test_df.collect()
                sample = sample_df[0]
                try:
                    input = ctx.populate_values(
                        transformed_column["input"], transformer["input"], preserve_column_refs=True
                    )
                except CortexException as e:
                    e.wrap("input")
                    raise
                transformer_input = create_transformer_inputs_from_map(input, sample)
                initial_transformed_value = trans_impl.transform_python(transformer_input)
                inferred_python_type = infer_python_type(initial_transformed_value)

                for row in sample_df:
                    transformer_input = create_transformer_inputs_from_map(input, row)
                    transformed_value = trans_impl.transform_python(transformer_input)
                    if inferred_python_type != infer_python_type(transformed_value):
                        raise UserException(
                            "transformed column " + column_name,
                            "type inference failed, mixed data types in dataframe.",
                            'expected type of "'
                            + transformed_sample
                            + '" to be '
                            + inferred_python_type,
                        )

                ctx.write_metadata(transformed_column["id"], {"type": inferred_python_type})

            transform_python_collect = execute_transform_python(
                column_name, test_df, ctx, spark, validate=True
            ).collect()
        except Exception as e:
            raise UserRuntimeException(
                "transformed column " + column_name,
                transformed_column["transformer"] + ".transform_python",
            ) from e

    if hasattr(trans_impl, "transform_spark"):
        try:
            transform_spark_df = execute_transform_spark(column_name, test_df, ctx, spark)

            # check that the return object is a dataframe
            if type(transform_spark_df) is not DataFrame:
                raise UserException(
                    "expected pyspark.sql.dataframe.DataFrame but got type {}".format(
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

            if transformer["output_type"] == consts.COLUMN_TYPE_INFERRED:
                inferred_spark_type = SPARK_TYPE_TO_CORTEX_TYPE[
                    transform_spark_df.select(column_name).schema[0].dataType
                ]
                ctx.write_metadata(transformed_column["id"], {"type": inferred_spark_type})

            # check that transformer run on data
            try:
                transform_spark_df.select(column_name).collect()
            except Exception as e:
                raise UserRuntimeException("function transform_spark") from e

            # check that expected output column has the correct data type
            if transformer["output_type"] != consts.COLUMN_TYPE_INFERRED:
                actual_structfield = transform_spark_df.select(column_name).schema.fields[0]
                if (
                    actual_structfield.dataType
                    not in CORTEX_TYPE_TO_ACCEPTABLE_SPARK_TYPES[transformer["output_type"]]
                ):
                    raise UserException(
                        "incorrect column type: expected {}, got {}.".format(
                            " or ".join(
                                str(t)
                                for t in CORTEX_TYPE_TO_ACCEPTABLE_SPARK_TYPES[
                                    transformer["output_type"]
                                ]
                            ),
                            actual_structfield.dataType,
                        )
                    )

            # perform the necessary casting for the column
            transform_spark_df = transform_spark_df.withColumn(
                column_name,
                F.col(column_name).cast(
                    CORTEX_TYPE_TO_SPARK_TYPE[ctx.get_inferred_column_type(column_name)]
                ),
            )

            # check that the function doesn't modify the schema of the other columns in the input dataframe
            if set(transform_spark_df.columns) - set([column_name]) != set(test_df.columns):
                logger.error("expected schema:")

                log_df_schema(test_df, logger.error)

                logger.error("found schema (with {} dropped):".format(column_name))
                log_df_schema(transform_spark_df.drop(column_name), logger.error)

                raise UserException(
                    "a column besides {} was modifed in the output dataframe".format(column_name)
                )
        except CortexException as e:
            raise UserRuntimeException(
                "transformed column " + column_name,
                transformed_column["transformer"] + ".transform_spark",
            ) from e

    if hasattr(trans_impl, "transform_spark") and hasattr(trans_impl, "transform_python"):
        if (
            transformer["output_type"] == consts.COLUMN_TYPE_INFERRED
            and inferred_spark_type != inferred_python_type
        ):
            raise UserException(
                "transformed column " + column_name,
                "type inference failed, transform_spark and transform_python had differing types.",
                "transform_python: " + inferred_python_type,
                "transform_spark: " + inferred_spark_type,
            )

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
    trans_impl, _ = ctx.get_transformer_impl(column_name)

    if hasattr(trans_impl, "transform_spark"):
        try:
            df = execute_transform_spark(column_name, df, ctx, spark)
            return df.withColumn(
                column_name,
                F.col(column_name).cast(
                    CORTEX_TYPE_TO_SPARK_TYPE[ctx.get_inferred_column_type(column_name)]
                ),
            )
        except CortexException as e:
            raise UserRuntimeException(
                "transformed column " + column_name,
                transformed_column["transformer"] + ".transform_spark",
            ) from e
    elif hasattr(trans_impl, "transform_python"):
        try:
            return execute_transform_python(column_name, df, ctx, spark)
        except Exception as e:
            raise UserRuntimeException(
                "transformed column " + column_name,
                transformed_column["transformer"] + ".transform_python",
            ) from e
    else:
        raise UserException(
            "transformed column " + column_name,
            "transformer " + transformed_column["transformer"],
            "transform_spark(), transform_python(), or both must be defined",
        )


def transform(model_name, accumulated_df, ctx, spark):
    model = ctx.models[model_name]
    column_names = ctx.extract_column_names(
        [model["input"], model["target_column"], model.get("training_input")]
    )

    for column_name in column_names:
        accumulated_df = transform_column(column_name, accumulated_df, ctx, spark)

    return accumulated_df
