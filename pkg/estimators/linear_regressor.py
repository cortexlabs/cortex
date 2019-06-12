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


import tensorflow as tf


def create_estimator(run_config, model_config):
    feature_columns = []

    for col_name in model_config["input"]["numeric_columns"]:
        feature_columns.append(tf.feature_column.numeric_column(col_name))

    for col_info in model_config["input"]["categorical_columns_with_vocab"]:
        col = tf.feature_column.categorical_column_with_vocabulary_list(
            col_info["col"], col_info["vocab"]
        )

        if "weight_column" in col_info:
            col = tf.feature_column.weighted_categorical_column(col, col_info["weight_column"])

        feature_columns.append(col)

    for col_info in model_config["input"]["categorical_columns_with_identity"]:
        col = tf.feature_column.categorical_columns_with_identity(
            col_info["col"], col_info["num_classes"]
        )

        if "weight_column" in col_info:
            col = tf.feature_column.weighted_categorical_column(col, col_info["weight_column"])

        feature_columns.append(col)

    for col_info in model_config["input"]["categorical_columns_with_hash_bucket"]:
        col = tf.feature_column.categorical_columns_with_hash_bucket(
            col_info["col"], col_info["hash_bucket_size"]
        )

        if "weight_column" in col_info:
            col = tf.feature_column.weighted_categorical_column(col, col_info["weight_column"])

        feature_columns.append(col)

    for col_info in model_config["input"]["bucketized_columns"]:
        feature_columns.append(
            tf.feature_column.bucketized_column(
                tf.feature_column.numeric_column(col_info["col"]), col_info["boundaries"]
            )
        )

    return tf.estimator.LinearRegressor(
        feature_columns=feature_columns,
        weight_column=model_config["input"].get("weight_column", None),
        config=run_config,
    )
