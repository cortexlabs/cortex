def aggregate_spark(data, columns, args):
    import pyspark.sql.functions as F

    max_review_length_row = (
        data.select(F.length(F.col(columns["col"])).alias("char_count"))
        .agg(F.max(F.col("char_count")).alias("max_char_length"))
        .collect()
    )

    return max_review_length_row[0]["max_char_length"]
