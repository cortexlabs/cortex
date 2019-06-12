import tensorflow as tf


def create_estimator(run_config, model_config):
    aggregates = model_config["input"]["aggregates"]

    feature_columns = [
        tf.feature_column.indicator_column(
            tf.feature_column.categorical_column_with_vocabulary_list(
                "sex", aggregates["sex_vocab"]
            )
        ),
        tf.feature_column.indicator_column(
            tf.feature_column.categorical_column_with_vocabulary_list(
                "smoker", aggregates["smoker_vocab"]
            )
        ),
        tf.feature_column.indicator_column(
            tf.feature_column.categorical_column_with_vocabulary_list(
                "region", aggregates["region_vocab"]
            )
        ),
        tf.feature_column.bucketized_column(
            tf.feature_column.numeric_column("age"), aggregates["age_buckets"]
        ),
        tf.feature_column.bucketized_column(
            tf.feature_column.numeric_column("bmi"), aggregates["bmi_buckets"]
        ),
        tf.feature_column.indicator_column(
            tf.feature_column.categorical_column_with_vocabulary_list(
                "children", aggregates["children_set"]
            )
        ),
    ]

    return tf.estimator.DNNRegressor(
        feature_columns=feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        config=run_config,
    )
