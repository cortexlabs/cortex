def aggregate_spark(data, columns, args):
    import pyspark.sql.functions as F
    from pyspark.ml.feature import StopWordsRemover, RegexTokenizer

    regexTokenizer = RegexTokenizer(inputCol=columns["col"], outputCol="token_list", pattern="\\W")
    regexTokenized = regexTokenizer.transform(data)

    remover = StopWordsRemover(inputCol="token_list", outputCol="filtered_word_list")
    vocab_rows = (
        remover.transform(regexTokenized)
        .select(F.explode(F.col("filtered_word_list")).alias("word"))
        .groupBy("word")
        .count()
        .orderBy(F.col("count").desc())
        .limit(args["vocab_size"])
        .select("word")
        .collect()
    )

    vocab = [row["word"] for row in vocab_rows]
    reverse_dict = {word: 2 + idx for idx, word in enumerate(vocab)}

    return {**reverse_dict}
