import tensorflow as tf


def create_estimator(run_config, model_config):
    feature_columns = []
    suits = [1, 2, 3, 4]
    ranks = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13]

    for feature_column in model_config["feature_columns"]:
        if feature_column["tags"]["type"] == "suit":
            categorical_column = tf.feature_column.categorical_column_with_vocabulary_list(
                feature_column["name"], suits
            )
            indicator_column = tf.feature_column.indicator_column(categorical_column)
            feature_columns.append(indicator_column)

        elif feature_column["tags"]["type"] == "rank":
            categorical_column = tf.feature_column.categorical_column_with_vocabulary_list(
                feature_column["name"], ranks
            )
            indicator_column = tf.feature_column.indicator_column(categorical_column)
            feature_columns.append(indicator_column)

    return tf.estimator.DNNClassifier(
        feature_columns=feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        n_classes=10,
        config=run_config,
    )
