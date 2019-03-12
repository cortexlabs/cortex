# Aggregates

Aggregate columns at scale.

## Config

```yaml
- kind: aggregate  # (required)
  name: <string>  # aggregate name (required)
  aggregator: <string>  # the name of the aggregator to use (required)
  inputs:
    columns:
      <string>: <string> or <[string]>  # map of column input name to raw column name(s) (required)
      ...
    args:
      <string>: <value>  # value may be a constant or literal value (optional)
      ...
  compute:
    executors: <int>  # number of spark executors (default: 1)
    driver_cpu: <string>  # CPU request for spark driver (default: 1)
    driver_mem: <string>  # memory request for spark driver (default: 500Mi)
    driver_mem_overhead: <string>  # off-heap (non-JVM) memory allocated to the driver (overrides mem_overhead_factor) (default: min[driver_mem * 0.4, 384Mi])
    executor_cpu: <string>  # CPU request for each spark executor (default: 1)
    executor_mem: <string>  # memory request for each spark executor (default: 500Mi)
    executor_mem_overhead: <string>  # off-heap (non-JVM) memory allocated to each executor (overrides mem_overhead_factor) (default: min[executor_mem * 0.4, 384Mi])
    mem_overhead_factor: <float>  # the proportion of driver_mem/executor_mem which will be additionally allocated for off-heap (non-JVM) memory (default: 0.4)
  tags:
    <string>: <scalar>  # arbitrary key/value pairs to attach to the resource (optional)
    ...
```

Note: the `columns` and `args` fields of the the aggregate must match the data types of the `columns` and `args` fields of the selected aggregator.

Each `args` value may be the name of a constant or a literal value. Any string value will be assumed to be the name of a constant. To use a string literal as an arg, escape it with double quotes (e.g. `arg_name: "\"string literal\""`.

See <!-- CORTEX_VERSION_MINOR -->[`aggregators.yaml`](https://github.com/cortexlabs/cortex/blob/0.2/pkg/aggregators/aggregators.yaml) for a list of built-in aggregators.

## Example

```yaml
- kind: aggregate
  name: age_bucket_boundaries
  aggregator: cortex.bucket_boundaries
  inputs:
    columns:
      col: age  # the name of a numeric raw column
    args:
      num_buckets: 5  # a value to be used as num_buckets

- kind: aggregate
  name: price_bucket_boundaries
  aggregator: cortex.bucket_boundaries
  inputs:
    columns:
      col: price  # the name of a numeric raw column
    args:
      num_buckets: num_buckets  # the name of an INT constant
```
