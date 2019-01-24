# Models

Cortex can train any model that implements the TensorFlow Estimator API. Models can be trained using any subset of the raw and transformed features.

## Implementation

```python
import tensorflow as tf

def create_estimator(run_config, model_config):
    """Create an estimator to train the model.

    Args:
        run_config: An instance of tf.estimator.RunConfig to be used when creating
            the estimator.

        model_config: The Cortex configuration for the model.
            Note: nested resources are expanded (e.g. model_config["target"])
            will be the configuration for the target feature, rather than the
            name of the target feature).

    Returns:
        An instance of tf.estimator.Estimator to train the model.
    """
    pass
```

## Example

```python
import tensorflow as tf

def create_estimator(run_config, model_config):
    columns = [
        tf.feature_column.numeric_column("sepal_length_normalized"),
        tf.feature_column.numeric_column("sepal_width_normalized"),
        tf.feature_column.numeric_column("petal_length_normalized"),
        tf.feature_column.numeric_column("petal_width_normalized"),
    ]

    return tf.estimator.DNNClassifier(
        feature_columns=columns,
        hidden_units=model_config["hparams"]["hidden_units"],
        n_classes=len(model_config["aggregates"]["class_index"]),
        config=run_config,
    )
```

## Pre-installed PyPI Packages

The following PyPI have been pre-installed and can be used in your implementations.

```text
numpy==1.15.4
pillow==5.4.1
pandas==0.23.4
scipy==1.2.0
sympy==1.3
statsmodels==0.9.0
python-dateutil==2.7.5
six==1.11.0
wrapt==1.11.0
requests==2.21.0
oauthlib==3.0.0
httplib2==0.12.0
```
