import tensorflow as tf


def create_estimator(run_config, model_config):
    feature_columns = [
        tf.feature_column.numeric_column(feature_column["name"])
        for feature_column in model_config["feature_columns"]
    ]

    return tf.estimator.DNNClassifier(
        feature_columns=feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        n_classes=2,
        weight_column="weight_column",
        config=run_config,
    )
