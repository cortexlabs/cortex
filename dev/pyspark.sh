#!/bin/bash
set -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

docker build $ROOT -f images/spark/Dockerfile -t spark-console:latest

docker run --rm -it --hostname=spark --name=spark-console --entrypoint=/opt/spark/bin/pyspark spark-console:latest

### Example dataframe creation ###
# df = spark.createDataFrame([(10,1.1),(12,2.2),(13,3.3)], ["int", "double"]).toDF("age", "years")

### Example reading/writing iris ###
# from pyspark.sql.types import *
# schema = StructType(
#     [
#         StructField(name="sepal_length", dataType=FloatType(), nullable=False),
#         StructField(name="sepal_width", dataType=FloatType(), nullable=False),
#         StructField(name="petal_length", dataType=FloatType(), nullable=False),
#         StructField(name="petal_width", dataType=FloatType(), nullable=False),
#         StructField(name="class", dataType=StringType(), nullable=False),
#     ]
# )
# df = spark.read.csv("s3a://cortex-examples/iris.csv", mode="FAILFAST", header=False, schema=schema)
# df.write.mode("overwrite").parquet("s3a://data-david/iris.parquet")
