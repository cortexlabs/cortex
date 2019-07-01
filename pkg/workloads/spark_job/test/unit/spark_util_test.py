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
from lib import util
import pytest
from pyspark.sql.types import *
from pyspark.sql import Row
import pyspark.sql.functions as F
from mock import MagicMock, call
from py4j.protocol import Py4JJavaError
from datetime import datetime

pytestmark = pytest.mark.usefixtures("spark")


def add_res_ref(input):
    return util.resource_escape_seq_raw + input


def test_read_csv_valid(spark, write_csv_file, ctx_obj, get_context):
    csv_str = "\n".join(["a,0.1,", "b,1,1", "c,1.1,4"])

    path_to_file = write_csv_file(csv_str)

    ctx_obj["environment"] = {
        "data": {
            "type": "csv",
            "path": path_to_file,
            "schema": [add_res_ref("a_str"), add_res_ref("b_float"), add_res_ref("c_long")],
        }
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "-"},
        "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": "-"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "-"},
    }

    assert spark_util.read_csv(get_context(ctx_obj), spark).count() == 3


def test_read_csv_invalid_type(spark, write_csv_file, ctx_obj, get_context):
    csv_str = "\n".join(["a,0.1,", "b,b,1", "c,1.1,4"])

    path_to_file = write_csv_file(csv_str)

    ctx_obj["environment"] = {
        "data": {
            "type": "csv",
            "path": path_to_file,
            "schema": [add_res_ref("a_str"), add_res_ref("b_long"), add_res_ref("c_long")],
        }
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "-"},
        "b_long": {"name": "b_long", "type": "INT_COLUMN", "required": True, "id": "-"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "-"},
    }

    with pytest.raises(UserException):
        spark_util.ingest(get_context(ctx_obj), spark).collect()


def test_read_csv_infer_type(spark, write_csv_file, ctx_obj, get_context):
    test_cases = [
        {
            "csv": ["a,0.1,", "b,0.1,1", "c,1.1,4"],
            "schema": [add_res_ref("a_str"), add_res_ref("b_float"), add_res_ref("c_long")],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INFERRED_COLUMN", "required": True, "id": "-"},
                "b_float": {
                    "name": "b_float",
                    "type": "INFERRED_COLUMN",
                    "required": True,
                    "id": "-",
                },
                "c_long": {
                    "name": "c_float",
                    "type": "INFERRED_COLUMN",
                    "required": True,
                    "id": "-",
                },
            },
            "expected_types": {"a_str": StringType(), "b_float": FloatType(), "c_long": LongType()},
        },
        {
            "csv": ["1,4,4.5", "1,3,1.2", "1,5,4.7"],
            "schema": [add_res_ref("a_str"), add_res_ref("b_int"), add_res_ref("c_float")],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "-"},
                "b_int": {"name": "b_int", "type": "INFERRED_COLUMN", "required": True, "id": "-"},
                "c_float": {
                    "name": "c_float",
                    "type": "INFERRED_COLUMN",
                    "required": True,
                    "id": "-",
                },
            },
            "expected_types": {"a_str": StringType(), "b_int": LongType(), "c_float": FloatType()},
        },
        {
            "csv": ["1,4,2017-09-16", "1,3,2017-09-16", "1,5,2017-09-16"],
            "schema": [add_res_ref("a_str"), add_res_ref("b_int"), add_res_ref("c_str")],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "-"},
                "b_int": {"name": "b_int", "type": "INFERRED_COLUMN", "required": True, "id": "-"},
                "c_str": {"name": "c_str", "type": "INFERRED_COLUMN", "required": True, "id": "-"},
            },
            "expected_types": {"a_str": StringType(), "b_int": LongType(), "c_str": StringType()},
        },
        {
            "csv": ["1,4,2017-09-16", "1,3,2017-09-16", "1,5,2017-09-16"],
            "schema": [add_res_ref("a_float"), add_res_ref("b_int"), add_res_ref("c_str")],
            "raw_columns": {
                "a_float": {"name": "a_float", "type": "FLOAT_COLUMN", "required": True, "id": "-"},
                "b_int": {"name": "b_int", "type": "INFERRED_COLUMN", "required": True, "id": "-"},
                "c_str": {"name": "c_str", "type": "INFERRED_COLUMN", "required": True, "id": "-"},
            },
            "expected_types": {"a_float": FloatType(), "b_int": LongType(), "c_str": StringType()},
        },
    ]

    for test in test_cases:
        csv_str = "\n".join(test["csv"])
        path_to_file = write_csv_file(csv_str)

        ctx_obj["environment"] = {
            "data": {"type": "csv", "path": path_to_file, "schema": test["schema"]}
        }

        ctx_obj["raw_columns"] = test["raw_columns"]

        df = spark_util.ingest(get_context(ctx_obj), spark)
        assert df.count() == len(test["expected_types"])
        inferred_col_type_map = {c.name: c.dataType for c in df.schema}
        for column_name in test["expected_types"]:
            assert inferred_col_type_map[column_name] == test["expected_types"][column_name]


def test_read_csv_infer_invalid(spark, write_csv_file, ctx_obj, get_context):
    test_cases = [
        {
            "csv": ["a,0.1,", "a,0.1,1", "a,1.1,4"],
            "schema": [add_res_ref("a_int"), add_res_ref("b_float"), add_res_ref("c_long")],
            "raw_columns": {
                "a_int": {"name": "a_int", "type": "INT_COLUMN", "required": True, "id": "-"},
                "b_float": {
                    "name": "b_float",
                    "type": "INFERRED_COLUMN",
                    "required": True,
                    "id": "-",
                },
                "c_long": {
                    "name": "c_long",
                    "type": "INFERRED_COLUMN",
                    "required": True,
                    "id": "-",
                },
            },
        },
        {
            "csv": ["a,1.1,", "a,1.1,1", "a,1.1,4"],
            "schema": [add_res_ref("a_str"), add_res_ref("b_int"), add_res_ref("c_int")],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INFERRED_COLUMN", "required": True, "id": "-"},
                "b_int": {"name": "b_int", "type": "INT_COLUMN", "required": True, "id": "-"},
                "c_int": {"name": "c_int", "type": "INFERRED_COLUMN", "required": True, "id": "-"},
            },
        },
    ]

    for test in test_cases:
        csv_str = "\n".join(test["csv"])
        path_to_file = write_csv_file(csv_str)

        ctx_obj["environment"] = {
            "data": {"type": "csv", "path": path_to_file, "schema": test["schema"]}
        }

        ctx_obj["raw_columns"] = test["raw_columns"]

        with pytest.raises(UserException):
            spark_util.ingest(get_context(ctx_obj), spark).collect()


def test_read_csv_missing_column(spark, write_csv_file, ctx_obj, get_context):
    csv_str = "\n".join(["a,1,", "b,1,"])

    path_to_file = write_csv_file(csv_str)

    ctx_obj["environment"] = {
        "data": {
            "type": "csv",
            "path": path_to_file,
            "schema": [add_res_ref("a_str"), add_res_ref("b_long"), add_res_ref("c_long")],
        }
    }

    ctx_obj["raw_columns"] = {
        "a_str": {"name": "a_str", "type": "STRING_COLUMN", "required": True, "id": "-"},
        "b_long": {"name": "b_long", "type": "INT_COLUMN", "required": True, "id": "-"},
        "c_long": {"name": "c_long", "type": "INT_COLUMN", "required": False, "id": "-"},
    }

    with pytest.raises(UserException) as exec_info:
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
            "schema": [add_res_ref("a_str"), add_res_ref("b_float"), add_res_ref("c_long")],
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
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
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


def test_ingest_parquet_infer_valid(spark, write_parquet_file, ctx_obj, get_context):
    tests = [
        {
            "data": [("a", 0.1, None), ("b", 1.0, None), ("c", 1.1, 4)],
            "schema": StructType(
                [
                    StructField("a_str", StringType()),
                    StructField("b_float", DoubleType()),
                    StructField("c_long", IntegerType()),
                ]
            ),
            "env": [
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
            ],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INFERRED_COLUMN", "required": True, "id": "1"},
                "b_float": {
                    "name": "b_float",
                    "type": "INFERRED_COLUMN",
                    "required": True,
                    "id": "2",
                },
                "c_long": {
                    "name": "c_long",
                    "type": "INFERRED_COLUMN",
                    "required": False,
                    "id": "3",
                },
            },
            "expected_types": [
                ("a_str", StringType()),
                ("b_float", FloatType()),
                ("c_long", LongType()),
            ],
        },
        {
            "data": [("1", 0.1, None), ("1", 1.0, None), ("1", 1.1, 4)],
            "schema": StructType(
                [
                    StructField("a_str", StringType()),
                    StructField("b_float", DoubleType()),
                    StructField("c_long", IntegerType()),
                ]
            ),
            "env": [
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
            ],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INFERRED_COLUMN", "required": True, "id": "1"},
                "b_float": {
                    "name": "b_float",
                    "type": "INFERRED_COLUMN",
                    "required": True,
                    "id": "2",
                },
                "c_long": {
                    "name": "c_long",
                    "type": "INFERRED_COLUMN",
                    "required": False,
                    "id": "3",
                },
            },
            "expected_types": [
                ("a_str", StringType()),
                ("b_float", FloatType()),
                ("c_long", LongType()),
            ],
        },
        {
            "data": [
                ("1", 0.1, datetime.now()),
                ("1", 1.0, datetime.now()),
                ("1", 1.1, datetime.now()),
            ],
            "schema": StructType(
                [
                    StructField("a_str", StringType()),
                    StructField("b_float", DoubleType()),
                    StructField("c_str", TimestampType()),
                ]
            ),
            "env": [
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_str", "raw_column": add_res_ref("c_str")},
            ],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INFERRED_COLUMN", "required": True, "id": "1"},
                "b_float": {
                    "name": "b_float",
                    "type": "INFERRED_COLUMN",
                    "required": True,
                    "id": "2",
                },
                "c_str": {"name": "c_str", "type": "INFERRED_COLUMN", "required": False, "id": "3"},
            },
            "expected_types": [
                ("a_str", StringType()),
                ("b_float", FloatType()),
                ("c_str", StringType()),
            ],
        },
        {
            "data": [
                ("1", [0.1, 12.0], datetime.now()),
                ("1", [1.23, 1.0], datetime.now()),
                ("1", [12.3, 1.1], datetime.now()),
            ],
            "schema": StructType(
                [
                    StructField("a_str", StringType()),
                    StructField("b_float", ArrayType(DoubleType()), True),
                    StructField("c_str", TimestampType()),
                ]
            ),
            "env": [
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_str", "raw_column": add_res_ref("c_str")},
            ],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INFERRED_COLUMN", "required": True, "id": "1"},
                "b_float": {
                    "name": "b_float",
                    "type": "FLOAT_LIST_COLUMN",
                    "required": True,
                    "id": "2",
                },
                "c_str": {"name": "c_str", "type": "INFERRED_COLUMN", "required": False, "id": "3"},
            },
            "expected_types": [
                ("a_str", StringType()),
                ("b_float", ArrayType(FloatType(), True)),
                ("c_str", StringType()),
            ],
        },
    ]

    for test in tests:
        data = test["data"]

        schema = test["schema"]

        path_to_file = write_parquet_file(spark, data, schema)

        ctx_obj["environment"] = {
            "data": {"type": "parquet", "path": path_to_file, "schema": test["env"]}
        }

        ctx_obj["raw_columns"] = test["raw_columns"]

        df = spark_util.ingest(get_context(ctx_obj), spark)

        assert df.count() == 3
        assert (
            sorted([(s.name, s.dataType) for s in df.schema], key=lambda x: x[0])
            == test["expected_types"]
        )


def test_read_parquet_infer_invalid(spark, write_parquet_file, ctx_obj, get_context):
    tests = [
        {
            "data": [("a", 0.1, None), ("b", 1.0, None), ("c", 1.1, 4)],
            "schema": StructType(
                [
                    StructField("a_str", StringType()),
                    StructField("b_float", DoubleType()),
                    StructField("c_long", IntegerType()),
                ]
            ),
            "env": [
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
            ],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INFERRED_COLUMN", "required": True, "id": "1"},
                "b_float": {"name": "b_float", "type": "INT_COLUMN", "required": True, "id": "2"},
                "c_long": {
                    "name": "c_long",
                    "type": "INFERRED_COLUMN",
                    "required": False,
                    "id": "3",
                },
            },
        },
        {
            "data": [("1", 0.1, "a"), ("1", 1.0, "a"), ("1", 1.1, "a")],
            "schema": StructType(
                [
                    StructField("a_str", StringType()),
                    StructField("b_float", DoubleType()),
                    StructField("c_str", StringType()),
                ]
            ),
            "env": [
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_str", "raw_column": add_res_ref("c_str")},
            ],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INFERRED_COLUMN", "required": True, "id": "1"},
                "b_float": {
                    "name": "b_float",
                    "type": "INFERRED_COLUMN",
                    "required": True,
                    "id": "2",
                },
                "c_str": {"name": "c_str", "type": "INT_COLUMN", "required": False, "id": "3"},
            },
        },
        {
            "data": [("a", 1, None), ("b", 1, None), ("c", 1, 4)],
            "schema": StructType(
                [
                    StructField("a_str", StringType()),
                    StructField("b_float", IntegerType()),
                    StructField("c_long", IntegerType()),
                ]
            ),
            "env": [
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
            ],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INFERRED_COLUMN", "required": True, "id": "1"},
                "b_float": {"name": "b_float", "type": "FLOAT_COLUMN", "required": True, "id": "2"},
                "c_long": {
                    "name": "c_long",
                    "type": "INFERRED_COLUMN",
                    "required": False,
                    "id": "3",
                },
            },
        },
        {
            "data": [("a", 1, None), ("b", 1, None), ("c", 1, 4)],
            "schema": StructType(
                [
                    StructField("a_str", StringType()),
                    StructField("b_float", IntegerType()),
                    StructField("c_long", IntegerType()),
                ]
            ),
            "env": [
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
            ],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INT_COLUMN", "required": True, "id": "1"},
                "b_float": {
                    "name": "b_float",
                    "type": "STRING_COLUMN",
                    "required": True,
                    "id": "2",
                },
                "c_long": {
                    "name": "c_long",
                    "type": "INFERRED_COLUMN",
                    "required": False,
                    "id": "3",
                },
            },
        },
        {
            "data": [("a", [1], None), ("b", [1], None), ("c", [1], 4)],
            "schema": StructType(
                [
                    StructField("a_str", StringType()),
                    StructField("b_float", ArrayType(IntegerType()), True),
                    StructField("c_long", IntegerType()),
                ]
            ),
            "env": [
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
            ],
            "raw_columns": {
                "a_str": {"name": "a_str", "type": "INT_COLUMN", "required": True, "id": "1"},
                "b_float": {
                    "name": "b_float",
                    "type": "FLOAT_LIST_COLUMN",
                    "required": True,
                    "id": "2",
                },
                "c_long": {
                    "name": "c_long",
                    "type": "INFERRED_COLUMN",
                    "required": False,
                    "id": "3",
                },
            },
        },
    ]

    for test in tests:
        data = test["data"]

        schema = test["schema"]

        path_to_file = write_parquet_file(spark, data, schema)

        ctx_obj["environment"] = {
            "data": {"type": "parquet", "path": path_to_file, "schema": test["env"]}
        }

        ctx_obj["raw_columns"] = test["raw_columns"]

        with pytest.raises(UserException) as exec_info:
            spark_util.ingest(get_context(ctx_obj), spark).collect()


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
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
                {"parquet_column_name": "d_long", "raw_column": add_res_ref("d_long")},
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
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
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
    assert "c_long" in str(exec_info.value) and "missing column" in str(exec_info.value)


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
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
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
    assert "c_long" in str(exec_info.value) and "type mismatch" in str(exec_info.value)


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
                {"parquet_column_name": "a_str", "raw_column": add_res_ref("a_str")},
                {"parquet_column_name": "b_float", "raw_column": add_res_ref("b_float")},
                {"parquet_column_name": "c_long", "raw_column": add_res_ref("c_long")},
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


def test_run_builtin_aggregators_success(spark, ctx_obj, get_context):
    ctx_obj["raw_columns"] = {"a": {"id": "2", "name": "a", "type": "INT_COLUMN"}}
    ctx_obj["aggregators"] = {
        "cortex.sum_int": {
            "name": "sum_int",
            "namespace": "cortex",
            "input": {"_type": "INT_COLUMN"},
            "output_type": "INT_COLUMN",
        }
    }
    ctx_obj["aggregates"] = {
        "sum_a": {
            "name": "sum_a",
            "id": "1",
            "aggregator": "cortex.sum_int",
            "input": add_res_ref("a"),
        }
    }
    aggregate_list = [v for v in ctx_obj["aggregates"].values()]

    ctx = get_context(ctx_obj)
    ctx.store_aggregate_result = MagicMock()
    ctx.populate_args = MagicMock(return_value={"ignorenulls": True})

    data = [Row(a=None), Row(a=1), Row(a=2), Row(a=3)]
    df = spark.createDataFrame(data, StructType([StructField("a", LongType())]))

    spark_util.run_builtin_aggregators(aggregate_list, df, ctx, spark)
    calls = [call(6, ctx_obj["aggregates"]["sum_a"])]
    ctx.store_aggregate_result.assert_has_calls(calls, any_order=True)


def test_infer_python_type():
    assert spark_util.infer_python_type(1) == consts.COLUMN_TYPE_INT
    assert spark_util.infer_python_type(1.0) == consts.COLUMN_TYPE_FLOAT
    assert spark_util.infer_python_type("cortex") == consts.COLUMN_TYPE_STRING

    assert spark_util.infer_python_type([1]) == consts.COLUMN_TYPE_INT_LIST
    assert spark_util.infer_python_type([1.0]) == consts.COLUMN_TYPE_FLOAT_LIST
    assert spark_util.infer_python_type(["cortex"]) == consts.COLUMN_TYPE_STRING_LIST
