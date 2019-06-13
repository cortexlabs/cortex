import tensorflow as tf


def create_estimator(run_config, model_config):
    feature_columns = [
        tf.feature_column.numeric_column(
            model_config["input"]["image_pixels"], shape=model_config["hparams"]["input_shape"]
        )
    ]

    return tf.estimator.DNNClassifier(
        feature_columns=feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        n_classes=10,
        config=run_config,
    )
