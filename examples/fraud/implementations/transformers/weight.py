def transform_spark(data, features, args, transformed_feature):
    import pyspark.sql.functions as F

    distribution = args["class_distribution"]

    return data.withColumn(
        transformed_feature,
        F.when(data[features["col"]] == 0, 1 - distribution[0]).otherwise(1 - distribution[1]),
    )
