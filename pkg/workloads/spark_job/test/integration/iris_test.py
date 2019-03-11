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
from lib.context import Context
from iris_context import raw_ctx

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
    cols_to_validate = [
        "9479e84647a126fe5ce36e6eeac35aacb7156cd8c8e0773e572a91a7f9c1e92",
        "690b9a1c2e717c7ec4304804d4d7fd54fba554d8ce4829062467a3dc4d5f0f8",
        "eb81ff65ce934e409ce18627cbb7d77c804289404fd62850fa5f915a1a9d87f",
        "98ee0c5e9935442ea77835297777f4ab916830db5cb1ec82590d8b03f53eb6c",
        "397a3c2785bcfdab244acdd11d65b415e3e4258b762deb8c17e600ce187c425",
    ]

    iris_data_string = "\n".join(",".join(str(val) for val in line) for line in iris_data)
    Path(os.path.join(str(local_storage_path), "iris.csv")).write_text(iris_data_string)

    raw_ctx["environment_data"]["csv_data"]["path"] = os.path.join(
        str(local_storage_path), "iris.csv"
    )

    ctx = Context(
        raw_obj=raw_ctx, cache_dir="/workspace/cache", local_storage_path=str(local_storage_path)
    )
    storage = ctx.storage

    raw_df = spark_job.ingest_raw_dataset(spark, ctx, cols_to_validate, should_ingest)

    assert raw_df.count() == 15
    assert storage.get_json(ctx.raw_dataset["metadata_key"])["dataset_size"] == 15
    for raw_column_id in cols_to_validate:
        path = os.path.join(raw_ctx["status_prefix"], raw_column_id, "jjd3l0fi4fhwqtgmpatg")
        status = storage.get_json(str(path))
        status["resource_id"] = raw_column_id
        status["exist_code"] = "succeeded"

    cols_to_aggregate = [
        "54ead5d565a57cad06972cc11d2f01f05c4e9e1dbfc525d1fa66b7999213722",
        "38159191e6018b929b42c7e73e8bfd19f5778bba79e84d9909f5d448ac15fc9",
        "986fd2cbc2b1d74aa06cf533b67d7dd7f54b5b7bf58689c58d0ec8c2568bae8",
        "317856401885874d95fffd349fe0878595e8c04833ba63c4546233ffd899e4d",
        "e7191b1effd1e4d351580f251aa35dc7c0b9825745b207fbb8cce904c94a937",
        "6a9481dc91eb3f82458356f1f5f98da6f25a69b460679e09a67988543f79e3f",
        "64594f51d3cfb55a3776d013102d5fdab29bfe7332ce0c4f7c916d64d3ca29f",
        "690f97171881c08770cac55137c672167a84324efba478cfd583ec98dd18844",
        "4deea2705f55fa8a38658546ea5c2d31e37d4aad43a874e091f1c1667b63a6e",
    ]
    spark_job.run_custom_aggregators(spark, ctx, cols_to_aggregate, raw_df)

    for aggregate_id in cols_to_aggregate:
        for aggregate_resource in raw_ctx["aggregates"].values():
            if aggregate_resource["id"] == aggregate_id:
                assert local_storage_path.joinpath(aggregate_resource["key"]).exists()
        path = os.path.join(raw_ctx["status_prefix"], aggregate_id, "jjd3l0fi4fhwqtgmpatg")
        status = storage.get_json(str(path))
        status["resource_id"] = aggregate_id
        status["exist_code"] = "succeeded"

    cols_to_transform = [
        "a44a0acbb54123d03d67b47469cf83712df2045b90aa99036dab99f37583d46",
        "41221f15eea0328c2987c44171f323529bfa7a196a697b1a87ff4915c143531",
        "360fe839dbc1ee1db2d0e0f0e8ca0d1a2cc54aed69e29843e0361d285ddb700",
        "7cbc111099c4bf38e27d6a05f9b2d37bdb9038f6f934be10298a718deae6db5",
        "6097e63c46b62b3cf70d86d9e1282bdd77d15d62bc4d132d9154bb5ddc1861d",
    ]

    spark_job.validate_transformers(spark, ctx, cols_to_transform, raw_df)

    for transformed_id in cols_to_transform:
        path = os.path.join(raw_ctx["status_prefix"], transformed_id, "jjd3l0fi4fhwqtgmpatg")
        status = storage.get_json(str(path))
        status["resource_id"] = transformed_id
        status["exist_code"] = "succeeded"

    training_datasets = ["5bdaecf9c5a0094d4a18df15348f709be8acfd3c6faf72c3f243956c3896e76"]

    spark_job.create_training_datasets(spark, ctx, training_datasets, raw_df)

    for dataset_id in training_datasets:
        path = os.path.join(raw_ctx["status_prefix"], transformed_id, "jjd3l0fi4fhwqtgmpatg")
        status = storage.get_json(str(path))
        status["resource_id"] = transformed_id
        status["exist_code"] = "succeeded"

        dataset = raw_ctx["models"]["dnn"]["dataset"]
        metadata_key = storage.get_json(dataset["metadata_key"])
        assert metadata_key["training_size"] + metadata_key["eval_size"] == 15
        assert local_storage_path.joinpath(dataset["train_key"], "_SUCCESS").exists()
        assert local_storage_path.joinpath(dataset["eval_key"], "_SUCCESS").exists()
