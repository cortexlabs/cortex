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
import math

import spark_util
import consts
from lib.exceptions import UserException

import pytest
from pyspark.sql.types import *
from pyspark.sql import Row
import pyspark.sql.functions as F
from mock import MagicMock, call
from py4j.protocol import Py4JJavaError


pytestmark = pytest.mark.usefixtures("spark")


def test_read_csv_valid(spark, write_csv_file, ctx_obj, get_context):
    csv_str = "\n".join(["a,0.1,", "b,1,1", "c,1.1,4"])

    path_to_file = write_csv_file(csv_str)

    ctx_obj["environment"] = {
        "data": {"type": "csv", "path": path_to_file, "schema": ["a_str", "b_float", "c_long"]}
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "-"},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": "-"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "-"},
    }

    assert spark_util.read_csv(get_context(ctx_obj), spark).count() == 3

    ctx_obj["environment"] = {
        "data": {
            "type": "csv",
            "path": path_to_file,
            "schema": ["a_str", "b_float", "c_long", "d_long"],
        }
    }

    assert spark_util.read_csv(get_context(ctx_obj), spark).count() == 3


def test_read_csv_invalid_type(spark, write_csv_file, ctx_obj, get_context):
    csv_str = "\n".join(["a,0.1,", "b,1,1", "c,1.1,4"])

    path_to_file = write_csv_file(csv_str)

    ctx_obj["environment"] = {
        "data": {"type": "csv", "path": path_to_file, "schema": ["a_str", "b_long", "c_long"]}
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "-"},
        "b_long": {"name": "b_long", "type": "INT_COLUMN", "required": True, "id": "-"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "-"},
    }

    with pytest.raises(Py4JJavaError):
        spark_util.ingest(get_context(ctx_obj), spark).collect()


def test_read_csv_missing_column(spark, write_csv_file, ctx_obj, get_context):
    csv_str = "\n".join(["a,0.1,", "b,1,1"])

    path_to_file = write_csv_file(csv_str)

    ctx_obj["environment"] = {
        "data": {"type": "csv", "path": path_to_file, "schema": ["a_str", "b_long", "c_long"]}
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "-"},
        "b_long": {"name": "b_long", "type": "INT_COLUMN", "required": True, "id": "-"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "-"},
    }

    with pytest.raises(Py4JJavaError) as exec_info:
        spark_util.ingest(get_context(ctx_obj), spark).collect()


def test_read_csv_valid_options(spark, write_csv_file, ctx_obj, get_context):
    csv_str = "\n".join(
        [
            "a_str|b_float|c_long",
            "   a   |1|",
            "|NaN|1",
            '"""weird"" having a | inside the string"|-Infini|NULL',
        ]
    )
    path_to_file = write_csv_file(csv_str)

    ctx_obj["environment"] = {
        "data": {
            "type": "csv",
            "path": path_to_file,
            "schema": ["a_str", "b_float", "c_long"],
            "csv_config": {
                "header": True,
                "sep": "|",
                "ignore_leading_white_space": False,
                "ignore_trailing_white_space": False,
                "nan_value": "NaN",
                "escape": '"',
                "negative_inf": "-Infini",
                "null_value": "NULL",
            },
        }
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "-"},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": "-"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "-"},
    }

    result_df = spark_util.read_csv(get_context(ctx_obj), spark)
    actual_results = result_df.select(*sorted(result_df.columns)).collect()

    assert len(actual_results) == 3
    assert actual_results[0] == Row(a_str="   a   ", b_float=float(1), c_long=None)
    assert actual_results[1].a_str == None
    assert math.isnan(
        actual_results[1].b_float
    )  # nan != nan so a row-wise comparison can't be done
    assert actual_results[1].c_long == 1
    assert actual_results[2] == Row(
        a_str='"weird" having a | inside the string', b_float=float("-Inf"), c_long=None
    )


def test_value_checker_required():
    raw_column_config = {"name": "a_str", "type": "STRING_COLUMN", "required": True}
    results = list(spark_util.value_checker(raw_column_config))
    results[0]["cond_col"] = str(results[0]["cond_col"]._jc)

    assert results == [
        {
            "col_name": "a_str",
            "cond_name": "required",
            "cond_col": str(F.col("a_str").isNull()._jc),
            "positive_cond_str": str(F.col("a_str").isNotNull()._jc),
        }
    ]


def test_value_checker_not_required():
    raw_column_config = {"name": "a_str", "type": "STRING_COLUMN", "required": False}
    results = list(spark_util.value_checker(raw_column_config))
    assert len(results) == 0


def test_value_checker_values():
    raw_column_config = {
        "name": "a_long",
        "type": "INT_COLUMN",
        "values": [1, 2, 3],
        "required": True,
    }
    results = sorted(spark_util.value_checker(raw_column_config), key=lambda x: x["cond_name"])
    for result in results:
        result["cond_col"] = str(result["cond_col"]._jc)

    assert results == [
        {
            "col_name": "a_long",
            "cond_name": "required",
            "cond_col": str(F.col("a_long").isNull()._jc),
            "positive_cond_str": str(F.col("a_long").isNotNull()._jc),
        },
        {
            "col_name": "a_long",
            "cond_name": "values",
            "cond_col": str((F.col("a_long").isin([1, 2, 3]) == False)._jc),
            "positive_cond_str": str(F.col("a_long").isin([1, 2, 3])._jc),
        },
    ]


def test_value_checker_min_max():
    raw_column_config = {"name": "a_long", "type": "INT_COLUMN", "min": 1, "max": 2}
    results = sorted(spark_util.value_checker(raw_column_config), key=lambda x: x["cond_name"])
    for result in results:
        result["cond_col"] = str(result["cond_col"]._jc)

    assert results == [
        {
            "col_name": "a_long",
            "cond_name": "max",
            "cond_col": ("(a_long > 2)"),
            "positive_cond_str": ("(a_long <= 2)"),
        },
        {
            "col_name": "a_long",
            "cond_name": "min",
            "cond_col": ("(a_long < 1)"),
            "positive_cond_str": ("(a_long >= 1)"),
        },
    ]


def test_value_check_data_valid(spark, ctx_obj, get_context):
    data = [("a", 0.1, None), ("b", 1.0, None), (None, 1.1, 0)]

    schema = StructType(
        [
            StructField("a_str", StringType()),
            StructField("b_float", FloatType()),
            StructField("c_long", LongType()),
        ]
    )

    df = spark.createDataFrame(data, schema)

    ctx_obj["raw_columns"] = {
        "a_str": {
            "name": "a_str",
            "type": "STRING_COLUMN",
            "required": False,
            "values": ["a", "b"],
            "id": 1,
        },
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": 2},
        "c_long": {
            "name": "c_long",
            "type": "INT_COLUMN",
            "required": False,
            "max": 1,
            "min": 0,
            "id": 3,
        },
    }

    ctx = get_context(ctx_obj)

    assert len(spark_util.value_check_data(ctx, df)) == 0


def test_value_check_data_invalid_null_value(spark, ctx_obj, get_context):
    data = [("a", None, None), ("b", 1.0, None), ("c", 1.1, 1)]

    schema = StructType(
        [
            StructField("a_str", StringType()),
            StructField("b_float", FloatType()),
            StructField("c_long", LongType()),
        ]
    )

    df = spark.createDataFrame(data, schema)

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": 1},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": 2},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "max": 1, "min": 0, "id": 3},
    }

    ctx = get_context(ctx_obj)
    validations = spark_util.value_check_data(ctx, df)
    assert validations == {"b_float": [("(b_float IS NOT NULL)", 1)]}


def test_value_check_data_invalid_out_of_range(spark, ctx_obj, get_context):
    data = [("a", 2.3, None), ("b", 1.0, None), ("c", 1.1, 4)]

    schema = StructType(
        [
            StructField("a_str", StringType()),
            StructField("b_float", FloatType()),
            StructField("c_long", LongType()),
        ]
    )

    df = spark.createDataFrame(data, schema)

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": 1},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": 2},
        "c_long": {
            "name": "c_long",
            "type": "INT_COLUMN",
            "required": False,
            "max": 1,
            "min": 0,
            "id": 3,
        },
    }

    ctx = get_context(ctx_obj)

    validations = spark_util.value_check_data(ctx, df)
    assert validations == {"c_long": [("(c_long <= 1)", 1)]}


def test_ingest_parquet_valid(spark, write_parquet_file, ctx_obj, get_context):
    data = [("a", 0.1, None), ("b", 1.0, None), ("c", 1.1, 4)]

    schema = StructType(
        [
            StructField("a_str", StringType()),
            StructField("b_float", DoubleType()),
            StructField("c_long", IntegerType()),
        ]
    )

    path_to_file = write_parquet_file(spark, data, schema)

    ctx_obj["environment"] = {
        "data": {
            "type": "parquet",
            "path": path_to_file,
            "schema": [
                {"parquet_column_name": "a_str", "raw_column_name": "a_str"},
                {"parquet_column_name": "b_float", "raw_column_name": "b_float"},
                {"parquet_column_name": "c_long", "raw_column_name": "c_long"},
            ],
        }
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "1"},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": "2"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "3"},
    }

    df = spark_util.ingest(get_context(ctx_obj), spark)

    assert df.count() == 3

    assert sorted([(s.name, s.dataType) for s in df.schema], key=lambda x: x[0]) == [
        ("a_str", StringType()),
        ("b_float", FloatType()),
        ("c_long", LongType()),
    ]


def test_ingest_parquet_extra_cols(spark, write_parquet_file, ctx_obj, get_context):
    data = [("a", 0.1, None), ("b", 1.0, None), ("c", 1.1, 4)]

    schema = StructType(
        [
            StructField("a_str", StringType()),
            StructField("b_float", FloatType()),
            StructField("c_long", LongType()),
        ]
    )

    path_to_file = write_parquet_file(spark, data, schema)

    ctx_obj["environment"] = {
        "data": {
            "type": "parquet",
            "path": path_to_file,
            "schema": [
                {"parquet_column_name": "a_str", "raw_column_name": "a_str"},
                {"parquet_column_name": "b_float", "raw_column_name": "b_float"},
                {"parquet_column_name": "c_long", "raw_column_name": "c_long"},
                {"parquet_column_name": "d_long", "raw_column_name": "d_long"},
            ],
        }
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "1"},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": "2"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "3"},
    }

    assert spark_util.ingest(get_context(ctx_obj), spark).count() == 3


def test_ingest_parquet_missing_cols(spark, write_parquet_file, ctx_obj, get_context):
    data = [("a", 0.1, None), ("b", 1.0, None), ("c", 1.1, 4)]

    schema = StructType(
        [
            StructField("a_str", StringType()),
            StructField("b_float", FloatType()),
            StructField("d_long", LongType()),
        ]
    )

    path_to_file = write_parquet_file(spark, data, schema)

    ctx_obj["environment"] = {
        "data": {
            "type": "parquet",
            "path": path_to_file,
            "schema": [
                {"parquet_column_name": "a_str", "raw_column_name": "a_str"},
                {"parquet_column_name": "b_float", "raw_column_name": "b_float"},
                {"parquet_column_name": "c_long", "raw_column_name": "c_long"},
            ],
        }
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "1"},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": "2"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "3"},
    }

    with pytest.raises(UserException) as exec_info:
        spark_util.ingest(get_context(ctx_obj), spark).collect()
    assert "c_long" in str(exec_info) and "missing column" in str(exec_info)


def test_ingest_parquet_type_mismatch(spark, write_parquet_file, ctx_obj, get_context):
    data = [("a", 0.1, None), ("b", 1.0, None), ("c", 1.1, 4.0)]

    schema = StructType(
        [
            StructField("a_str", StringType()),
            StructField("b_float", FloatType()),
            StructField("c_long", FloatType()),
        ]
    )

    path_to_file = write_parquet_file(spark, data, schema)

    ctx_obj["environment"] = {
        "data": {
            "type": "parquet",
            "path": path_to_file,
            "schema": [
                {"parquet_column_name": "a_str", "raw_column_name": "a_str"},
                {"parquet_column_name": "b_float", "raw_column_name": "b_float"},
                {"parquet_column_name": "c_long", "raw_column_name": "c_long"},
            ],
        }
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "1"},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": "2"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "3"},
    }

    with pytest.raises(UserException) as exec_info:
        spark_util.ingest(get_context(ctx_obj), spark).collect()
    assert "c_long" in str(exec_info) and "type mismatch" in str(exec_info)


def test_ingest_parquet_failed_requirements(
    capsys, spark, write_parquet_file, ctx_obj, get_context
):
    data = [(None, 0.1, None), ("b", 1.0, None), ("c", 1.1, 4)]

    schema = StructType(
        [
            StructField("a_str", StringType()),
            StructField("b_float", FloatType()),
            StructField("c_long", LongType()),
        ]
    )

    path_to_file = write_parquet_file(spark, data, schema)

    ctx_obj["environment"] = {
        "data": {
            "type": "parquet",
            "path": path_to_file,
            "schema": [
                {"parquet_column_name": "a_str", "raw_column_name": "a_str"},
                {"parquet_column_name": "b_float", "raw_column_name": "b_float"},
                {"parquet_column_name": "c_long", "raw_column_name": "c_long"},
            ],
        }
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "values": ["a", "b"], "id": "1"},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": "2"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "3"},
    }

    ctx = get_context(ctx_obj)
    df = spark_util.ingest(ctx, spark)

    validations = spark_util.value_check_data(ctx, df)
    assert validations == {"a_str": [("(a_str IN (a, b))", 1)]}


def test_column_names_to_index():
    sample_columns_input_config = {"b": "b_col", "a": "a_col"}
    actual_list, actual_dict = spark_util.column_names_to_index(sample_columns_input_config)
    assert (["a_col", "b_col"], {"b": 1, "a": 0}) == (actual_list, actual_dict)

    sample_columns_input_config = {"a": "a_col"}

    actual_list, actual_dict = spark_util.column_names_to_index(sample_columns_input_config)
    assert (["a_col"], {"a": 0}) == (actual_list, actual_dict)

    sample_columns_input_config = {"nums": ["a_long", "a_col", "b_col", "b_col"], "a": "a_long"}

    expected_col_list = ["a_col", "a_long", "b_col"]
    expected_columns_input_config = {"nums": [1, 0, 2, 2], "a": 1}
    actual_list, actual_dict = spark_util.column_names_to_index(sample_columns_input_config)

    assert (expected_col_list, expected_columns_input_config) == (actual_list, actual_dict)


def test_run_builtin_aggregators_success(spark, ctx_obj, get_context):
    ctx_obj["aggregators"] = {
        "cortex.sum": {"name": "sum", "namespace": "cortex"},
        "cortex.first": {"name": "first", "namespace": "cortex"},
    }
    ctx_obj["aggregates"] = {
        "sum_a": {
            "name": "sum_a",
            "id": "1",
            "aggregator": "cortex.sum",
            "inputs": {"columns": {"col": "a"}},
        },
        "first_a": {
            "id": "2",
            "name": "first_a",
            "aggregator": "cortex.first",
            "inputs": {"columns": {"col": "a"}, "args": {"ignorenulls": "some_constant"}},
        },
    }
    aggregate_list = [v for v in ctx_obj["aggregates"].values()]

    ctx = get_context(ctx_obj)
    ctx.store_aggregate_result = MagicMock()
    ctx.populate_args = MagicMock(return_value={"ignorenulls": True})

    data = [Row(a=None), Row(a=1), Row(a=2), Row(a=3)]
    df = spark.createDataFrame(data, StructType([StructField("a", LongType())]))

    spark_util.run_builtin_aggregators(aggregate_list, df, ctx, spark)
    calls = [call(6, ctx_obj["aggregates"]["sum_a"]), call(1, ctx_obj["aggregates"]["first_a"])]
    ctx.store_aggregate_result.assert_has_calls(calls, any_order=True)

    ctx.populate_args.assert_called_once_with({"ignorenulls": "some_constant"})


def test_run_builtin_aggregators_error(spark, ctx_obj, get_context):
    ctx_obj["aggregators"] = {"cortex.first": {"name": "first", "namespace": "cortex"}}
    ctx_obj["aggregates"] = {
        "first_a": {
            "name": "first_a",
            "aggregator": "cortex.first",
            "inputs": {
                "columns": {"col": "a"},
                "args": {"ignoreNulls": "some_constant"},  # supposed to be ignorenulls
            },
            "id": "1",
        }
    }

    aggregate_list = [v for v in ctx_obj["aggregates"].values()]

    ctx = get_context(ctx_obj)
    ctx.store_aggregate_result = MagicMock()
    ctx.populate_args = MagicMock(return_value={"ignoreNulls": True})

    data = [Row(a=None), Row(a=1), Row(a=2), Row(a=3)]
    df = spark.createDataFrame(data, StructType([StructField("a", LongType())]))

    with pytest.raises(Exception) as exec_info:
        spark_util.run_builtin_aggregators(aggregate_list, df, ctx, spark)

    ctx.store_aggregate_result.assert_not_called()
    ctx.populate_args.assert_called_once_with({"ignoreNulls": "some_constant"})


def test_infer_type():
    assert spark_util.infer_type(1) == consts.COLUMN_TYPE_INT
    assert spark_util.infer_type(1.0) == consts.COLUMN_TYPE_FLOAT
    assert spark_util.infer_type("cortex") == consts.COLUMN_TYPE_STRING

    assert spark_util.infer_type([1]) == consts.COLUMN_TYPE_INT_LIST
    assert spark_util.infer_type([1.0]) == consts.COLUMN_TYPE_FLOAT_LIST
    assert spark_util.infer_type(["cortex"]) == consts.COLUMN_TYPE_STRING_LIST
