import tensorflow as tf


def create_estimator(run_config, model_config):
    vocab_size = len(model_config["aggregates"]["reviews_vocab"])
    feature_column = tf.feature_column.categorical_column_with_identity(
        "embedding_input", vocab_size
    )
    return tf.estimator.LinearClassifier(feature_columns=[feature_column], config=run_config)
