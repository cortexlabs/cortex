# Aggregators

## Implementation

```python
def aggregate_spark(data, columns, args):
    """Aggregate a column in a PySpark context.

    This function is required.

    Args:
        data: A dataframe including all of the raw columns.

        columns: A dict with the same structure as the aggregator's input
            columns specifying the names of the dataframe's columns that
            contain the input columns.

        args: A dict with the same structure as the aggregator's input args
            containing the values of the args.

    Returns:
        Any json-serializable object that matches the data type of the aggregator.
    """
    pass
```

## Example

```python
def aggregate_spark(data, columns, args):
    from pyspark.ml.feature import QuantileDiscretizer

    discretizer = QuantileDiscretizer(
        numBuckets=args["num_buckets"], inputCol=columns["col"], outputCol="_"
    ).fit(data)

    return discretizer.getSplits()
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
