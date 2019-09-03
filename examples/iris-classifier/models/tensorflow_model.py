# sources copied/modified from https://github.com/tensorflow/models/blob/master/samples/core/get_started/

import tensorflow as tf
from sklearn import datasets, metrics
from sklearn.metrics import classification_report, confusion_matrix
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split


keys = ["sepal_length", "sepal_width", "petal_length", "petal_width"]

iris = load_iris()

X, y = iris.data, iris.target
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.8, random_state=42)

feature_columns = []
for key in keys:
    feature_columns.append(tf.feature_column.numeric_column(key=key))


def train_input_fn(features, labels, batch_size):
    irises = {}
    for idx, key in enumerate(keys):
        irises[key] = features[:, idx]
    dataset = tf.data.Dataset.from_tensor_slices((irises, labels))
    dataset = dataset.shuffle(1000).repeat().batch(batch_size)

    return dataset


def eval_input_fn(features, labels, batch_size):
    irises = {}
    for idx, key in enumerate(keys):
        irises[key] = features[:, idx]
    if labels is None:
        inputs = irises
    else:
        inputs = (irises, labels)

    dataset = tf.data.Dataset.from_tensor_slices(inputs)
    dataset = dataset.batch(batch_size)
    return dataset


classifier = tf.estimator.DNNClassifier(
    feature_columns=feature_columns, hidden_units=[10, 10], n_classes=3
)

classifier.train(input_fn=lambda: train_input_fn(X_train, y_train, 100), steps=1000)
eval_result = classifier.evaluate(input_fn=lambda: eval_input_fn(X_test, y_test, 100))
print("\nTest set accuracy: {accuracy:0.3f}\n".format(**eval_result))


def json_serving_input_fn():
    placeholders = {}
    features = {}
    for key in keys:
        placeholders[key] = tf.placeholder(shape=[None], dtype=tf.float64, name=key)
        features[key] = tf.expand_dims(placeholders[key], -1)
    return tf.estimator.export.ServingInputReceiver(
        features=features, receiver_tensors=placeholders
    )


classifier.export_savedmodel("export", json_serving_input_fn, strip_default_attrs=True)
