import nltk
from nltk.corpus import stopwords

nltk.download("stopwords")


def aggregate_spark(data, columns, args):
    from pyspark.ml.feature import StopWordsRemover, RegexTokenizer
    import pyspark.sql.functions as F
    from pyspark.sql.types import IntegerType

    regexTokenizer = RegexTokenizer(inputCol=columns["col"], outputCol="token_list", pattern="\\W")
    regexTokenized = regexTokenizer.transform(data)

    remover = StopWordsRemover(
        inputCol="token_list",
        outputCol="filtered_word_list",
        stopWords=set(stopwords.words("english")),
    )
    max_review_length_row = (
        remover.transform(regexTokenized)
        .select(F.size(F.col("filtered_word_list")).alias("word_count"))
        .agg(F.max(F.col("word_count")).alias("max_review_length"))
        .collect()
    )

    return max_review_length_row[0]["max_review_length"]
