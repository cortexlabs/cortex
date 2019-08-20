# Aggregators

## Implementation

```python
def aggregate_spark(data, input):
    """Aggregate a column in a PySpark context.

    This function is required.

    Args:
        data: A dataframe including all of the raw columns.

        input: The aggregate's input object. Column references in the input are
            replaced by their names (e.g. "@column1" will be replaced with "column1"),
            and all other resource references (e.g. constants) are replaced by their
            runtime values.

    Returns:
        Any serializable object that matches the output type of the aggregator.
    """
    pass
```

See Cortex's built-in <!-- CORTEX_VERSION_MINOR -->[aggregators](https://github.com/cortexlabs/cortex/blob/0.7/pkg/aggregators) for example implementations.

## Example

```python
def aggregate_spark(data, input):
    from pyspark.ml.feature import QuantileDiscretizer

    discretizer = QuantileDiscretizer(
        numBuckets=input["num_buckets"], inputCol=input["col"], outputCol="_"
    ).fit(data)

    return discretizer.getSplits()
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
pillow==6.1.0
```

You can install additional PyPI packages and import your own Python packages. See [Python Packages](python-packages.md) for more details.
