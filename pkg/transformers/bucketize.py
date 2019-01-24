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


def transform_spark(data, features, args, transformed_feature):
    from pyspark.ml.feature import Bucketizer
    import pyspark.sql.functions as F

    new_b = Bucketizer(
        splits=args["bucket_boundaries"], inputCol=features["num"], outputCol=transformed_feature
    )
    return new_b.transform(data).withColumn(
        transformed_feature, F.col(transformed_feature).cast("int")
    )


def transform_python(sample, args):
    num = sample["num"]
    buckets = args["bucket_boundaries"][1:]
    for id, v in enumerate(buckets):
        if num < v:
            return id
