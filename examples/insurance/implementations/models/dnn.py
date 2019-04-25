import tensorflow as tf


def create_estimator(run_config, model_config):
    feature_columns = []
    for param in model_config["inputs"]["categorical"]:
        feature_columns.append(
            tf.feature_column.indicator_column(
                tf.feature_column.categorical_column_with_vocabulary_list(
                    param["feature"], param["categories"]
                )
            )
        )

    for param in model_config["inputs"]["bucketized"]:
        feature_columns.append(
            tf.feature_column.bucketized_column(
                tf.feature_column.numeric_column(param["feature"], param["buckets"])
            )
        )

    for param in model_config["inputs"]["numeric"]:
        feature_columns.append(tf.feature_column.numeric_column(param))

    return tf.estimator.DNNRegressor(
        feature_columns=feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        config=run_config,
    )
