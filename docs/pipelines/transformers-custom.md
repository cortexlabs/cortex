# Transformers

Transformers run when transforming data before model training and when responding to prediction requests. You may define transformers for both a PySpark and a Python context. The PySpark implementation is optional but recommended for large-scale data processing.

## Implementation

```python
def transform_spark(data, input, transformed_column_name):
    """Transform a column in a PySpark context.

    This function is optional (recommended for large-scale data processing).

    Args:
        data: A dataframe including all of the raw columns.

        input: The transformed column's input object. Column references in the input are
            replaced by their names (e.g. "@column1" will be replaced with "column1"),
            and all other resource references (e.g. constants and aggregates) are replaced
            by their runtime values.

        transformed_column_name: The name of the column containing the transformed
            data that is to be appended to the dataframe.

    Returns:
        The original 'data' dataframe with an added column named <transformed_column_name>
        which contains the transformed data.
    """
    pass


def transform_python(input):
    """Transform a single data sample outside of a PySpark context.

    This function is required for any columns that are used during inference.

    Args:
        input: The transformed column's input object. Column references in the input are
            replaced by their values in the sample (e.g. "@column1" will be replaced with
            the value for column1), and all other resource references (e.g. constants and
            aggregates) are replaced by their runtime values.

    Returns:
        The transformed value.
    """
    pass


def reverse_transform_python(transformed_value, input):
    """Reverse transform a single data sample outside of a PySpark context.

    This function is optional, and only relevant for certain one-to-one
    transformers.

    Args:
        transformed_value: The transformed data value.

        input: The transformed column's input object. Column references in the input are
            replaced by their names (e.g. "@column1" will be replaced with "column1"),
            and all other resource references (e.g. constants and aggregates) are replaced
            by their runtime values.

    Returns:
        The raw data value that corresponds to the transformed value.
    """
    pass
```

See Cortex's built-in <!-- CORTEX_VERSION_MINOR -->[transformers](https://github.com/cortexlabs/cortex/blob/0.5/pkg/transformers) for example implementations.

## Example

```python
def transform_spark(data, input, transformed_column_name):
    return data.withColumn(
        transformed_column_name, ((data[input["col"]] - input["mean"]) / input["stddev"])
    )

def transform_python(input):
    return (input["col"] - input["mean"]) / input["stddev"]

def reverse_transform_python(transformed_value, input):
    return input["mean"] + (transformed_value * input["stddev"])
```

## Pre-installed Packages

The following packages have been pre-installed and can be used in your implementations:

```text
pyspark==2.4.2
boto3==1.9.78
msgpack==0.6.1
numpy>=1.13.3,<2
requirements-parser==0.2.0
packaging==19.0.0
```

You can install additional PyPI packages and import your own Python packages. See [Python Packages](../advanced/python-packages.md) for more details.
