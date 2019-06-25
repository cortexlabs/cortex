import tensorflow as tf


def create_estimator(run_config, model_config):
    embedding_feature_columns = []
    for feature_col_data in model_config["input"]["embedding_columns"]:
        embedding_col = tf.feature_column.embedding_column(
            tf.feature_column.categorical_column_with_identity(
                feature_col_data["col"], len(feature_col_data["vocab"]["index"])
            ),
            model_config["hparams"]["embedding_size"],
        )
        embedding_feature_columns.append(embedding_col)

    return tf.estimator.DNNRegressor(
        feature_columns=embedding_feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        config=run_config,
    )
