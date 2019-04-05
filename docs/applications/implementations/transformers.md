# Transformers

Transformers run both when transforming data before model training and when responding to prediction requests. You may define transformers for both a PySpark and a Python context. The PySpark implementation is optional but recommended for large-scale data processing.

## Implementation

```python
def transform_spark(data, columns, args, transformed_column_name):
    """Transform a column in a PySpark context.

    This function is optional (recommended for large-scale data processing).

    Args:
        data: A dataframe including all of the raw columns.

        columns: A dict with the same structure as the transformer's input
            columns specifying the names of the dataframe's columns that
            contain the input columns.

        args: A dict with the same structure as the transformer's input args
            containing the runtime values of the args.

        transformed_column_name: The name of the column containing the transformed
            data that is to be appended to the dataframe.

    Returns:
        The original 'data' dataframe with an added column named <transformed_column_name>
        which contains the transformed data.
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
def transform_spark(data, columns, args, transformed_column_name):
    return data.withColumn(
        transformed_column_name, ((data[columns["num"]] - args["mean"]) / args["stddev"])
    )

def transform_python(sample, args):
    return (sample["num"] - args["mean"]) / args["stddev"]

def reverse_transform_python(transformed_value, args):
    return args["mean"] + (transformed_value * args["stddev"])
```

## Pre-installed Packages

The following packages have been pre-installed and can be used in your implementations:

```text
pyspark==2.4.1
boto3==1.9.78
msgpack==0.6.1
numpy>=1.13.3,<2
requirements-parser==0.2.0
packaging==19.0.0
```

You can install additional PyPI packages and import your own Python packages. See [Python Packages](../advanced/python-packages.md) for more details.
