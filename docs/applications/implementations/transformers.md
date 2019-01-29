# Transformers

Transformers run both when transforming features before model training and when responding to prediction requests. You may define transformers for both a PySpark and a Python context. The PySpark implementation is optional but recommended for large-scale feature processing.

## Implementation

```python
def transform_spark(data, features, args, transformed_feature):
    """Transform a feature in a PySpark context.

    This function is optional (recommended for large-scale feature processing).

    Args:
        data: A dataframe including all of the raw features.

        features: A dict with the same structure as the transformer's input
            features specifying the names of the dataframe's columns that
            contain the input features.

        args: A dict with the same structure as the transformer's input args
            containing the runtime values of the args.

        transformed_feature: The name of the column containing the transformed
            data that is to be appended to the dataframe.

    Returns:
        The original 'data' dataframe with an added column with the name of the
        transformed_feature arg containing the transformed data.
    """
    pass


def transform_python(sample, args):
    """Transform a single data sample outside of a PySpark context.

    This function is required.

    Args:
        sample: A dict with the same structure as the transformer's input
            features containing a data sample to transform.

        args: A dict with the same structure as the transformer's input args
            containing the runtime values of the args.

    Returns:
        The transformed value.
    """
    pass


def reverse_transform_python(transformed_value, args):
    """Reverse transform a single data sample outside of a PySpark context.

    This function is optional, and only relevant for certain one-to-one
    transformers.

    Args:
        transformed_value: The transformed data value.

        args: A dict with the same structure as the transformer's input args
            containing the runtime values of the args.

    Returns:
        The raw data value that corresponds to the transformed value.
    """
    pass
```

## Example

```python
def transform_spark(data, features, args, transformed_feature):
    return data.withColumn(
        transformed_feature, ((data[features["num"]] - args["mean"]) / args["stddev"])
    )

def transform_python(sample, args):
    return (sample["num"] - args["mean"]) / args["stddev"]

def reverse_transform_python(transformed_value, args):
    return args["mean"] + (transformed_value * args["stddev"])
```

## Pre-installed PyPI Packages

The following PyPI have been pre-installed and can be used in your implementations.

```text
pyyaml==3.13
numpy==1.15.4
pillow==5.4.1
pandas==0.23.4
scipy==1.2.0
sympy==1.3
statsmodels==0.9.0
python-dateutil==2.7.5
six==1.11.0
wrapt==1.11.0
```
