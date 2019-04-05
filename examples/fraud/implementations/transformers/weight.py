def transform_spark(data, columns, args, transformed_column_name):
    import pyspark.sql.functions as F

    distribution = args["class_distribution"]

    return data.withColumn(
        transformed_column_name,
        F.when(data[columns["col"]] == 0, distribution[1]).otherwise(distribution[0]),
    )
