import tensorflow as tf
from tensorflow import keras


def create_estimator(run_config, model_config):
    hparams = model_config["hparams"]
    vocab_size = len(model_config["input"]["vocab"])

    def model_fn(features, labels, mode, params):
        embedding_input = features["embedding_input"]
        model = keras.Sequential()
        model.add(keras.layers.Embedding(vocab_size, 16))
        model.add(keras.layers.GlobalAveragePooling1D())
        model.add(keras.layers.Dense(16, activation=tf.nn.relu))
        model.add(keras.layers.Dense(2))

        if mode is tf.estimator.ModeKeys.PREDICT:
            logits = model(embedding_input, training=False)
            probabilities = tf.nn.softmax(logits, -1)
            prediction = tf.argmax(probabilities, axis=-1, output_type=tf.int32)
            predictions = {"class_ids": prediction, "probabilities": probabilities}
            return tf.estimator.EstimatorSpec(
                mode=tf.estimator.ModeKeys.PREDICT,
                predictions=predictions,
                export_outputs={"predict": tf.estimator.export.PredictOutput(predictions)},
            )

        if mode is tf.estimator.ModeKeys.TRAIN:
            optimizer = tf.train.AdamOptimizer(learning_rate=hparams["learning_rate"])
            logits = model(embedding_input, training=True)
            loss = tf.losses.sparse_softmax_cross_entropy(labels=labels, logits=logits)
            return tf.estimator.EstimatorSpec(
                mode=tf.estimator.ModeKeys.TRAIN,
                loss=loss,
                train_op=optimizer.minimize(loss, tf.train.get_or_create_global_step()),
            )

        if mode is tf.estimator.ModeKeys.EVAL:
            logits = model(embedding_input, training=False)
            loss = tf.losses.sparse_softmax_cross_entropy(labels=labels, logits=logits)
            return tf.estimator.EstimatorSpec(
                mode=tf.estimator.ModeKeys.EVAL,
                loss=loss,
                eval_metric_ops={
                    "acc": tf.metrics.accuracy(
                        labels=labels, predictions=tf.argmax(logits, axis=-1)
                    )
                },
            )

    estimator = tf.estimator.Estimator(model_fn=model_fn, config=run_config)
    return estimator
