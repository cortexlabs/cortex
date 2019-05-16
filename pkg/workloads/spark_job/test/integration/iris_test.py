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
from spark_job import spark_job
from lib.exceptions import UserException
from lib import Context
from test.integration import iris_context

import pytest
from pyspark.sql.types import *
from pyspark.sql import Row
import pyspark.sql.functions as F
from mock import MagicMock, call
from py4j.protocol import Py4JJavaError
from pathlib import Path
import os

pytestmark = pytest.mark.usefixtures("spark")


iris_data = [
    [5.1, 3.5, 1.4, 0.2, "Iris-setosa"],
    [4.9, 3.0, 1.4, 0.2, "Iris-setosa"],
    [4.7, 3.2, 1.3, 0.2, "Iris-setosa"],
    [4.6, 3.1, 1.5, 0.2, "Iris-setosa"],
    [5.0, 3.6, 1.4, 0.2, "Iris-setosa"],
    [7.0, 3.2, 4.7, 1.4, "Iris-versicolor"],
    [6.4, 3.2, 4.5, 1.5, "Iris-versicolor"],
    [6.9, 3.1, 4.9, 1.5, "Iris-versicolor"],
    [5.5, 2.3, 4.0, 1.3, "Iris-versicolor"],
    [6.5, 2.8, 4.6, 1.5, "Iris-versicolor"],
    [6.3, 3.3, 6.0, 2.5, "Iris-virginica"],
    [5.8, 2.7, 5.1, 1.9, "Iris-virginica"],
    [7.1, 3.0, 5.9, 2.1, "Iris-virginica"],
    [6.3, 2.9, 5.6, 1.8, "Iris-virginica"],
    [6.5, 3.0, 5.8, 2.2, "Iris-virginica"],
]


def test_simple_end_to_end(spark):
    local_storage_path = Path("/workspace/local_storage")
    local_storage_path.mkdir(parents=True, exist_ok=True)
    should_ingest = True
    input_data_path = os.path.join(str(local_storage_path), "iris.csv")

    raw_ctx = iris_context.get(input_data_path)

    workload_id = raw_ctx["raw_columns"]["raw_float_columns"]["sepal_length"]["workload_id"]

    cols_to_validate = []

    for column_type in raw_ctx["raw_columns"].values():
        for raw_column in column_type.values():
            cols_to_validate.append(raw_column["id"])

    iris_data_string = "\n".join(",".join(str(val) for val in line) for line in iris_data)
    Path(os.path.join(str(local_storage_path), "iris.csv")).write_text(iris_data_string)

    ctx = Context(
        raw_obj=raw_ctx, cache_dir="/workspace/cache", local_storage_path=str(local_storage_path)
    )
    storage = ctx.storage

    raw_df = spark_job.ingest_raw_dataset(spark, ctx, cols_to_validate, should_ingest)

    assert raw_df.count() == 15
    assert ctx.get_metadata("raw_dataset")["dataset_size"] == 15
    for raw_column_id in cols_to_validate:
        path = os.path.join(raw_ctx["status_prefix"], raw_column_id, workload_id)
        status = storage.get_json(str(path))
        status["resource_id"] = raw_column_id
        status["exist_code"] = "succeeded"

    cols_to_aggregate = [r["id"] for r in raw_ctx["aggregates"].values()]

    spark_job.run_custom_aggregators(spark, ctx, cols_to_aggregate, raw_df)

    for aggregate_id in cols_to_aggregate:
        for aggregate_resource in raw_ctx["aggregates"].values():
            if aggregate_resource["id"] == aggregate_id:
                assert local_storage_path.joinpath(aggregate_resource["key"]).exists()
        path = os.path.join(raw_ctx["status_prefix"], aggregate_id, workload_id)
        status = storage.get_json(str(path))
        status["resource_id"] = aggregate_id
        status["exist_code"] = "succeeded"

    cols_to_transform = [r["id"] for r in raw_ctx["transformed_columns"].values()]
    spark_job.validate_transformers(spark, ctx, cols_to_transform, raw_df)

    for transformed_id in cols_to_transform:
        path = os.path.join(raw_ctx["status_prefix"], transformed_id, workload_id)
        status = storage.get_json(str(path))
        status["resource_id"] = transformed_id
        status["exist_code"] = "succeeded"

    training_datasets = [raw_ctx["models"]["dnn"]["dataset"]["id"]]

    spark_job.create_training_datasets(spark, ctx, training_datasets, raw_df)

    for dataset_id in training_datasets:
        path = os.path.join(raw_ctx["status_prefix"], transformed_id, workload_id)
        status = storage.get_json(str(path))
        status["resource_id"] = transformed_id
        status["exist_code"] = "succeeded"

        dataset = raw_ctx["models"]["dnn"]["dataset"]
        metadata = ctx.get_metadata("training_datasets", "dnn")
        assert metadata["training_size"] + metadata["eval_size"] == 15
        assert local_storage_path.joinpath(dataset["train_key"], "_SUCCESS").exists()
        assert local_storage_path.joinpath(dataset["eval_key"], "_SUCCESS").exists()
