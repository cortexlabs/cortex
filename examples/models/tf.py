# sources copied/modified from https://github.com/tensorflow/models/blob/master/samples/core/get_started/

import tensorflow as tf
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
import shutil
import os

EXPORT_DIR = "iris_tf_export"

def input_fn(features, labels, batch_size, mode):
    """An input function for training"""
    dataset = tf.data.Dataset.from_tensor_slices((features, labels))
    if mode == tf.estimator.ModeKeys.TRAIN:
        dataset = dataset.shuffle(1000).repeat()
    dataset = dataset.batch(batch_size)
    dataset_it = dataset.make_one_shot_iterator()
    irises, labels = dataset_it.get_next()
    return {"irises": irises}, labels


def json_serving_input_fn():
    inputs = tf.placeholder(shape=[4], dtype=tf.float64)
    features = {"irises": tf.expand_dims(inputs, 0)}
    return tf.estimator.export.ServingInputReceiver(features=features, receiver_tensors=inputs)


def my_model(features, labels, mode, params):
    """DNN with three hidden layers and learning_rate=0.1."""
    net = features["irises"]
    for units in params["hidden_units"]:
        net = tf.layers.dense(net, units=units, activation=tf.nn.relu)

    logits = tf.layers.dense(net, params["n_classes"], activation=None)

    predicted_classes = tf.argmax(logits, 1)
    if mode == tf.estimator.ModeKeys.PREDICT:
        predictions = {
            "class_ids": predicted_classes[:, tf.newaxis],
            "probabilities": tf.nn.softmax(logits),
            "logits": logits,
        }
        return tf.estimator.EstimatorSpec(
            mode=mode,
            predictions=predictions,
            export_outputs={
                "predict": tf.estimator.export.PredictOutput(
                    {
                        "class_ids": predicted_classes[:, tf.newaxis],
                        "probabilities": tf.nn.softmax(logits),
                    }
                )
            },
        )

    loss = tf.losses.sparse_softmax_cross_entropy(labels=labels, logits=logits)

    accuracy = tf.metrics.accuracy(labels=labels, predictions=predicted_classes, name="acc_op")
    metrics = {"accuracy": accuracy}
    tf.summary.scalar("accuracy", accuracy[1])

    if mode == tf.estimator.ModeKeys.EVAL:
        return tf.estimator.EstimatorSpec(mode, loss=loss, eval_metric_ops=metrics)

    optimizer = tf.train.AdagradOptimizer(learning_rate=0.1)
    train_op = optimizer.minimize(loss, global_step=tf.train.get_global_step())
    return tf.estimator.EstimatorSpec(mode, loss=loss, train_op=train_op)


iris = load_iris()
X, y = iris.data, iris.target
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.8, random_state=42)

classifier = tf.estimator.Estimator(
    model_fn=my_model, model_dir=EXPORT_DIR, params={"hidden_units": [10, 10], "n_classes": 3}
)


train_input_fn = lambda: input_fn(X_train, y_train, 100, tf.estimator.ModeKeys.TRAIN)
eval_input_fn = lambda: input_fn(X_test, y_test, 100, tf.estimator.ModeKeys.EVAL)
serving_input_fn = lambda: json_serving_input_fn()
exporter = tf.estimator.FinalExporter("estimator", serving_input_fn, as_text=False)
train_spec = tf.estimator.TrainSpec(train_input_fn, max_steps=1000)
eval_spec = tf.estimator.EvalSpec(eval_input_fn, exporters=[exporter], name="estimator-eval")

tf.estimator.train_and_evaluate(classifier, train_spec, eval_spec)

# exported path looks like iris_tf_export/export/estimator/1562353043/variables
# need to zip the versioned dir
estimator_dir = EXPORT_DIR + "/export/estimator"
shutil.make_archive("tensorflow", "zip", os.path.join(estimator_dir))

# clean up
shutil.rmtree(EXPORT_DIR)
