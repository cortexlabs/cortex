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
import os

import pytest
from pyspark.sql.types import *
from pyspark.sql import Row
import pyspark.sql.functions as F
from mock import MagicMock, call
from py4j.protocol import Py4JJavaError
from pathlib import Path

from cortex.spark_job import spark_util
from cortex.spark_job import spark_job
from cortex.lib.exceptions import UserException
from cortex.lib import Context
from cortex.spark_job.test.integration import insurance_context


pytestmark = pytest.mark.usefixtures("spark")


insurance_data = [
    [19, "female", 27.9, 0, "yes", "southwest", 16884.924],
    [18, "male", 33.77, 1, "no", "southeast", 1725.5523],
    [28, "male", 33, 3, "no", "southeast", 4449.462],
    [33, "male", 22.705, 0, "no", "northwest", 21984.47061],
    [32, "male", 28.88, 0, "no", "northwest", 3866.8552],
    [31, "female", 25.74, 0, "no", "southeast", 3756.6216],
    [46, "female", 33.44, 1, "no", "southeast", 8240.5896],
    [37, "female", 27.74, 3, "no", "northwest", 7281.5056],
    [37, "male", 29.83, 2, "no", "northeast", 6406.4107],
    [60, "female", 25.84, 0, "no", "northwest", 28923.13692],
    [25, "male", 26.22, 0, "no", "northeast", 2721.3208],
    [62, "female", 26.29, 0, "yes", "southeast", 27808.7251],
    [23, "male", 34.4, 0, "no", "southwest", 1826.843],
    [56, "female", 39.82, 0, "no", "southeast", 11090.7178],
    [27, "male", 42.13, 0, "yes", "southeast", 39611.7577],
]


def test_simple_end_to_end(spark):
    local_storage_path = Path("/workspace/local_storage")
    local_storage_path.mkdir(parents=True, exist_ok=True)
    should_ingest = True
    input_data_path = os.path.join(str(local_storage_path), "insurance.csv")

    raw_ctx = insurance_context.get(input_data_path)

    workload_id = raw_ctx["raw_columns"]["raw_string_columns"]["smoker"]["workload_id"]

    cols_to_validate = []

    for column_type in raw_ctx["raw_columns"].values():
        for raw_column in column_type.values():
            cols_to_validate.append(raw_column["id"])

    insurance_data_string = "\n".join(",".join(str(val) for val in line) for line in insurance_data)
    Path(os.path.join(str(local_storage_path), "insurance.csv")).write_text(insurance_data_string)

    ctx = Context(
        raw_obj=raw_ctx, cache_dir="/workspace/cache", local_storage_path=str(local_storage_path)
    )
    storage = ctx.storage

    raw_df = spark_job.ingest_raw_dataset(spark, ctx, cols_to_validate, should_ingest)

    assert raw_df.count() == 15
    assert ctx.get_metadata(ctx.raw_dataset["key"])["dataset_size"] == 15
    for raw_column_id in cols_to_validate:
        path = os.path.join(raw_ctx["status_prefix"], raw_column_id, workload_id)
        status = storage.get_json(str(path))
        status["resource_id"] = raw_column_id
        status["exist_code"] = "succeeded"

    cols_to_aggregate = [r["id"] for r in raw_ctx["aggregates"].values()]

    spark_job.run_aggregators(spark, ctx, cols_to_aggregate, raw_df)

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
        metadata = ctx.get_metadata(dataset["id"])
        assert metadata["training_size"] + metadata["eval_size"] == 15
        assert local_storage_path.joinpath(dataset["train_key"], "_SUCCESS").exists()
        assert local_storage_path.joinpath(dataset["eval_key"], "_SUCCESS").exists()
