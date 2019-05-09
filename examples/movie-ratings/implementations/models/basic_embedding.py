import tensorflow as tf


def create_estimator(run_config, model_config):
    user_id_index = model_config["aggregates"]["user_id_index"]
    movie_id_index = model_config["aggregates"]["movie_id_index"]

    feature_columns = [
        tf.feature_column.embedding_column(
            tf.feature_column.categorical_column_with_identity(
                "user_id_indexed", len(user_id_index)
            ),
            model_config["hparams"]["embedding_size"],
        ),
        tf.feature_column.embedding_column(
            tf.feature_column.categorical_column_with_identity(
                "movie_id_indexed", len(movie_id_index)
            ),
            model_config["hparams"]["embedding_size"],
        ),
    ]

    return tf.estimator.DNNRegressor(
        feature_columns=feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        config=run_config,
    )
