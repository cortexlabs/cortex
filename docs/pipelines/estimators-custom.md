# Estimators

Cortex can train any model that implements the TensorFlow Estimator API. Models can be trained using any subset of the raw and transformed columns.

## Implementation

```python
import tensorflow as tf

def create_estimator(run_config, model_config):
    """Create an estimator to train the model.

    Args:
        run_config: An instance of tf.estimator.RunConfig to be used when creating
            the estimator.

        model_config: The Cortex configuration for the model. Column references in all
            inputs (i.e. model_config["target_column"], model_config["input"], and
            model_config["training_input"]) are replaced by their names (e.g. "@column1"
            will be replaced with "column1"). All other resource references (e.g. constants
            and aggregates) are replaced by their runtime values.

    Returns:
        An instance of tf.estimator.Estimator to train the model.
    """
    pass
```

See the [tf.estimator.RunConfig](https://www.tensorflow.org/api_docs/python/tf/estimator/RunConfig) and [tf.estimator.Estimator](https://www.tensorflow.org/api_docs/python/tf/estimator/Estimator) documentation for more information.

See Cortex's built-in <!-- CORTEX_VERSION_MINOR -->[estimators](https://github.com/cortexlabs/cortex/blob/0.7/pkg/estimators) for example implementations.

## Example

```python
import tensorflow as tf

def create_estimator(run_config, model_config):
    feature_columns = []
    for col_name in model_config["input"]["numeric_columns"]:
        feature_columns.append(tf.feature_column.numeric_column(col_name))

    return tf.estimator.DNNClassifier(
        feature_columns=feature_columns,
        n_classes=model_config["input"]["num_classes"],
        hidden_units=model_config["hparams"]["hidden_units"],
        config=run_config,
    )
```

## Pre-installed Packages

You can import PyPI packages or your own Python packages to help create more complex models. See [Python Packages](python-packages.md) for more details.

The following packages have been pre-installed and can be used in your implementations:

```text
tensorflow==1.14.0
boto3==1.9.78
msgpack==0.6.1
numpy>=1.13.3,<2
requirements-parser==0.2.0
packaging==19.0.0
pillow==6.1.0
regex==2017.4.5
requests==2.21.0
```

You can install additional PyPI packages and import your own Python packages. See [Python Packages](python-packages.md) for more details.


# Tensorflow Transformations
You can preprocess input features and labels to your model by defining a `transform_tensorflow` function. You can define tensor transformations you want to apply to the features and labels tensors before they are passed to the model.

## Implementation

```python
def transform_tensorflow(features, labels, model_config):
     """Define tensor transformations for the feature and label tensors.

    Args:
        features: A feature dictionary of column names to feature tensors.

        labels: The label tensor.

        model_config: The Cortex configuration for the model. Column references in all
            inputs (i.e. model_config["target_column"], model_config["input"], and
            model_config["training_input"]) are replaced by their names (e.g. "@column1"
            will be replaced with "column1"). All other resource references (e.g. constants
            and aggregates) are replaced by their runtime values.

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
