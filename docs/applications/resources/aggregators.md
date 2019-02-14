# Aggregators

An aggregator converts a set of columns and arbitrary values into a single value. Each aggregator has an input schema and an output data type. The input schema is a map which specifies the name and data type of each input column and argument. Aggregators run before transformers.

## Config

```yaml
- kind: aggregator  # (required)
  name: <string>  # aggregator name (required)
  path: <string>  # path to the implementation file, relative to the application root (default: implementations/aggregators/<name>.py)
  output_type: <value_type>  # output data type (required)
  inputs:
    columns:
      <string>: <input_column_type>  # map of column input name to column input type(s) (required)
      ...
    args:
      <string>: <value_type>  # map of arg input name to value input type(s) (optional)
      ...
```

See [Data Types](data-types.md) for a list of valid data types.

## Example

```yaml
- kind: aggregator
  name: bucket_boundaries
  output_type: [FLOAT]
  inputs:
    columns:
      num: FLOAT_COLUMN|INT_COLUMN
    args:
      num_buckets: INT
```

## Built-in Aggregators

Cortex includes common aggregators that can be used out of the box (see <!-- CORTEX_VERSION_MINOR -->[`aggregators.yaml`](https://github.com/cortexlabs/cortex/blob/0.1/pkg/aggregators/aggregators.yaml)). To use built-in aggregators, use the `cortex` namespace in the aggregator name (e.g. `cortex.normalize`).
