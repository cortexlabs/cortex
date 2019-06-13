# Aggregators

An aggregator converts a set of columns and arbitrary values into a single value. Each aggregator has an input type and an output type. Aggregators run before transformers.

Custom aggregators can be implemented in Python or PySpark. See the [implementation docs](../implementations/aggregators.md) for a detailed guide.

## Config

```yaml
- kind: aggregator
  name: <string>  # aggregator name (required)
  path: <string>  # path to the implementation file, relative to the application root (default: implementations/aggregators/<name>.py)
  output_type: <output_type>  # the output type of the aggregator (required)
  input: <input_type>  # the input type of the aggregator (required)
```

See [Data Types](data-types.md) for details about input and output types.

## Example

```yaml
- kind: aggregator
  name: bucket_boundaries
  path: bucket_boundaries.py
  output_type: [FLOAT]
  input:
    num: FLOAT_COLUMN|INT_COLUMN
    num_buckets: INT
```

## Built-in Aggregators

Cortex includes common aggregators that can be used out of the box (see <!-- CORTEX_VERSION_MINOR -->[`aggregators.yaml`](https://github.com/cortexlabs/cortex/blob/0.4/pkg/aggregators/aggregators.yaml)). To use built-in aggregators, use the `cortex` namespace in the aggregator name (e.g. `cortex.mean`).
