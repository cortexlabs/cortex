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

""" pytest fixtures that can be resued across tests. the filename needs to be conftest.py
"""

import logging
import pytest
import uuid
import os

from pyspark import SparkConf
from pyspark.sql import SparkSession
from lib.context import Context
import consts
import shutil


def quiet_py4j():
    """ turn down spark logging for the test context """
    logger = logging.getLogger("py4j")
    logger.setLevel(logging.WARN)


@pytest.fixture(scope="session")
def spark(request):
    """ fixture for creating a spark context
    Args:
        request: pytest.FixtureRequest object
    """
    spark = (
        SparkSession.builder.master("local[2]")
        .appName("pytest-pyspark-local-testing")
        .getOrCreate()
    )
    request.addfinalizer(lambda: spark.stop())

    quiet_py4j()
    return spark


class NoneDict(dict):
    def __getitem__(self, key):
        return dict.get(self, key)


@pytest.fixture(scope="function", name="ctx_obj")
def empty_context_obj():
    return NoneDict(
        app=NoneDict(),
        cortex_config=NoneDict(api_version=consts.CORTEX_VERSION, region=""),
        raw_columns={},
        transformed_columns={},
        aggregates={},
        constants={},
        aggregators={},
        models={},
        apis={},
    )


@pytest.fixture(scope="module")
def get_context():
    def _get_context(d):
        return Context(obj=d, cache_dir=".")

    return _get_context


@pytest.fixture(scope="function")
def write_csv_file(request):
    def _write_csv_file(csv_string, path="."):
        filename = str(uuid.uuid4()) + ".csv"
        path_to_file = os.path.join(path, filename)
        with open(path_to_file, "w") as f:
            f.write(csv_string)

        request.addfinalizer(lambda: os.remove(path_to_file))

        return path_to_file

    return _write_csv_file


@pytest.fixture(scope="function")
def write_parquet_file(request):
    def _write_parquet_file(spark, tuple_list, schema, path="."):
        table_name = str(uuid.uuid4())
        path_to_parquet_table = os.path.join(path, table_name)

        spark.createDataFrame(tuple_list, schema).write.parquet(path_to_parquet_table)

        request.addfinalizer(lambda: shutil.rmtree(path_to_parquet_table))

        return path_to_parquet_table

    return _write_parquet_file
