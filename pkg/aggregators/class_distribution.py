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


def aggregate_spark(data, features, args):
    import pyspark.sql.functions as F
    from functools import reduce

    rows = data.groupBy(F.col(features["col"])).count().orderBy(F.col("count").desc()).collect()

    sum = float(reduce(lambda x, y: x + y, (r[1] for r in rows)))

    return {r[0]: (r[1] / sum) for r in rows}
