# Aggregators

## Implementation

```python
def aggregate_spark(data, features, args):
    """Aggregate a feature in a PySpark context.

    This function is required.

    Args:
        data: A dataframe including all of the raw features.

        features: A dict with the same structure as the aggregator's input
            features specifying the names of the dataframe's columns that
            contain the input features.

        args: A dict with the same structure as the aggregator's input args
            containing the values of the args.

    Returns:
        Any json-serializable object that matches the data type of the aggregator.
    """
    pass
```

## Example

```python
def aggregate_spark(data, features, args):
    from pyspark.ml.feature import QuantileDiscretizer

    discretizer = QuantileDiscretizer(
        numBuckets=args["num_buckets"], inputCol=features["col"], outputCol="_"
    ).fit(data)

    return discretizer.getSplits()
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

You can install additional PyPI packages and import your own Python packages. See [Custom Packages](../advanced/custom-packages.md) for more details.