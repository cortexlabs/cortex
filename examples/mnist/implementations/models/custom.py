import tensorflow as tf


def create_estimator(run_config, model_config):
    hparams = model_config["hparams"]

    def model_fn(features, labels, mode, params):
        x = features["image_pixels"]
        for i, feature_count in enumerate(hparams["hidden_units"]):
            with tf.variable_scope("layer_%d" % i):
                if hparams["layer_type"] == "conv":
                    x = tf.layers.conv2d(x, feature_count, hparams["kernel_size"], padding="SAME")
                elif hparams["layer_type"] == "dense":
                    x = tf.layers.dense(x, feature_count)

        x = tf.layers.flatten(x)
        with tf.variable_scope("logit"):
            x = tf.layers.dense(x, hparams["output_shape"][0], use_bias=False)

        probabilities = tf.nn.softmax(x, -1)
        prediction = tf.argmax(probabilities, axis=-1, output_type=tf.int32)
        if mode is tf.estimator.ModeKeys.PREDICT:
            return tf.estimator.EstimatorSpec(
                mode=mode,
                predictions=prediction,
                export_outputs={
                    "predict": tf.estimator.export.PredictOutput(
                        {"class_ids": prediction, "probabilities": probabilities}
                    )
                },
            )

        loss = tf.losses.sparse_softmax_cross_entropy(labels=labels, logits=x)
        lr = tf.constant(hparams["learning_rate"])
        optimizer = tf.train.GradientDescentOptimizer(lr)
        train_op = tf.contrib.layers.optimize_loss(
            name="training",
            loss=loss,
            global_step=tf.train.get_global_step(),
            learning_rate=lr,
            optimizer=optimizer,
            colocate_gradients_with_ops=True,
        )

        spec = tf.estimator.EstimatorSpec(
            mode,
            eval_metric_ops={"acc": tf.metrics.accuracy(labels=labels, predictions=prediction)},
            loss=loss,
            train_op=train_op,
        )
        return spec

    estimator = tf.estimator.Estimator(model_fn=model_fn, config=run_config)
    return estimator


def transform_tensorflow(features, labels, model_config):
    hparams = model_config["hparams"]

    features["image_pixels"] = tf.reshape(features["image_pixels"], hparams["input_shape"])
    return features, labels
