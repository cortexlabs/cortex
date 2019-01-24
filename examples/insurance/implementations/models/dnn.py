import tensorflow as tf


def create_estimator(run_config, model_config):
    feature_columns = [
        tf.feature_column.indicator_column(
            tf.feature_column.categorical_column_with_vocabulary_list("sex", ["female", "male"])
        ),
        tf.feature_column.indicator_column(
            tf.feature_column.categorical_column_with_vocabulary_list("smoker", ["yes", "no"])
        ),
        tf.feature_column.indicator_column(
            tf.feature_column.categorical_column_with_vocabulary_list(
                "region", ["northwest", "northeast", "southwest", "southeast"]
            )
        ),
        tf.feature_column.bucketized_column(
            tf.feature_column.numeric_column("age"), [15, 20, 25, 35, 40, 45, 50, 55, 60, 65]
        ),
        tf.feature_column.bucketized_column(
            tf.feature_column.numeric_column("bmi"), [15, 20, 25, 35, 40, 45, 50, 55]
        ),
        tf.feature_column.indicator_column(
            tf.feature_column.categorical_column_with_vocabulary_list(
                "children", model_config["aggregates"]["children_set"]
            )
        ),
    ]

    return tf.estimator.DNNRegressor(
        feature_columns=feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        config=run_config,
    )
