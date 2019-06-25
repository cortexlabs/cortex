def transform_spark(data, input, transformed_column_name):
    import pyspark.sql.functions as F

    distribution = input["class_distribution"]

    return data.withColumn(
        transformed_column_name,
        F.when(data[input["col"]] == 0, distribution[1]).otherwise(distribution[0]),
    )
