def aggregate_spark(data, columns, args):
    from pyspark.ml.feature import RegexTokenizer
    import pyspark.sql.functions as F
    from pyspark.sql.types import IntegerType

    regexTokenizer = RegexTokenizer(inputCol=columns["col"], outputCol="token_list", pattern="\\W")
    regexTokenized = regexTokenizer.transform(data)

    max_review_length_row = (
        regexTokenized.select(F.size(F.col("token_list")).alias("word_count"))
        .agg(F.max(F.col("word_count")).alias("max_review_length"))
        .collect()
    )

    return max_review_length_row[0]["max_review_length"]
