# Transformers

Transformers run both when transforming data before model training and when responding to prediction requests. You may define transformers for both a PySpark and a Python context. The PySpark implementation is optional but recommended for large-scale data processing.

## Implementation

```python
def transform_spark(data, columns, args, transformed_column):
    """Transform a column in a PySpark context.

    This function is optional (recommended for large-scale data processing).

    Args:
        data: A dataframe including all of the raw columns.

        columns: A dict with the same structure as the transformer's input
            columns specifying the names of the dataframe's columns that
            contain the input columns.

        args: A dict with the same structure as the transformer's input args
            containing the runtime values of the args.

        transformed_column: The name of the column containing the transformed
            data that is to be appended to the dataframe.

    Returns:
        The original 'data' dataframe with an added column with the name of the
        transformed_column arg containing the transformed data.
    """
    pass


def transform_python(sample, args):
    """Transform a single data sample outside of a PySpark context.

    This function is required.

    Args:
        sample: A dict with the same structure as the transformer's input
            columns containing a data sample to transform.

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
def transform_spark(data, columns, args, transformed_column):
    return data.withColumn(
        transformed_column, ((data[columns["num"]] - args["mean"]) / args["stddev"])
    )

def transform_python(sample, args):
    return (sample["num"] - args["mean"]) / args["stddev"]

def reverse_transform_python(transformed_value, args):
    return args["mean"] + (transformed_value * args["stddev"])
```

## Pre-installed Packages

The following packages have been pre-installed and can be used in your implementations:

```text
pyspark==2.4.0
pyyaml==3.13
numpy==1.15.4
pandas==0.23.4
scipy==1.2.0
sympy==1.3
statsmodels==0.9.0
python-dateutil==2.7.5
six==1.11.0
wrapt==1.11.0
```

You can install additional PyPI packages and import your own Python packages. See [Python Packages](../advanced/python-packages.md) for more details.