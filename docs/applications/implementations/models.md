# Models

Cortex can train any model that implements the TensorFlow Estimator API. Models can be trained using any subset of the raw and transformed columns.

## Implementation

```python
import tensorflow as tf

def create_estimator(run_config, model_config):
    """Create an estimator to train the model.

    Args:
        run_config: An instance of tf.estimator.RunConfig to be used when creating
            the estimator.

        model_config: The Cortex configuration for the model.
            Note: nested resources are expanded (e.g. model_config["target_column"])
            will be the configuration for the target column, rather than the
            name of the target column).

    Returns:
        An instance of tf.estimator.Estimator to train the model.
    """
    pass
```

## Example

```python
import tensorflow as tf

def create_estimator(run_config, model_config):
    feature_columns = [
        tf.feature_column.numeric_column("sepal_length_normalized"),
        tf.feature_column.numeric_column("sepal_width_normalized"),
        tf.feature_column.numeric_column("petal_length_normalized"),
        tf.feature_column.numeric_column("petal_width_normalized"),
    ]

    return tf.estimator.DNNClassifier(
        feature_columns=feature_columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        n_classes=len(model_config["aggregates"]["class_index"]),
        config=run_config,
    )
```

## Pre-installed Packages

You can import PyPI packages or your own Python packages to help create more complex models. See [Python Packages](../advanced/python-packages.md) for more details.

The following packages have been pre-installed and can be used in your implementations:

```text
tensorflow==1.12.0
boto3==1.9.78
msgpack==0.6.1
numpy>=1.13.3,<2
requirements-parser==0.2.0
packaging==19.0.0
```

You can install additional PyPI packages and import your own Python packages. See [Python Packages](../advanced/python-packages.md) for more details.


# Tensorflow Transformations
You can preprocess input features and labels to your model by defining a `transform_tensorflow` function. You can define tensor transformations you want to apply to the features and labels tensors before they are passed to the model.

## Implementation

```python
def transform_tensorflow(features, labels, model_config):
     """Define tensor transformations for the feature and label tensors.

    Args:
        features: A feature dictionary of column names to feature tensors.

        labels: The label tensor.

        model_config: The Cortex configuration for the model.
            Note: nested resources are expanded (e.g. model_config["target_column"])
            will be the configuration for the target column, rather than the
            name of the target column).


    Returns:
        features and labels tensors.
    """
    return features, labels
```

## Example

```python
import tensorflow as tf

def transform_tensorflow(features, labels, model_config):
    hparams = model_config["hparams"]
    features["image_pixels"] = tf.reshape(features["image_pixels"], hparams["input_shape"])
    return features, labels
```
