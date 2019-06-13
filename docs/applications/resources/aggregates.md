# Aggregates

Aggregate columns at scale.

## Config

```yaml
- kind: aggregate
  name: <string>  # aggregate name (required)
  aggregator: <string>  # the name of the aggregator to use (this or aggregator_path must be specified)
  aggregator_path: <string>  # a path to an aggregator implementation file (this or aggregator must be specified)
  input: <input_value>  # the input to the aggregator, which may contain references to columns and constants (e.g. @column1) (required)
  compute:
    executors: <int>  # number of spark executors (default: 1)
    driver_cpu: <string>  # CPU request for spark driver (default: 1)
    driver_mem: <string>  # memory request for spark driver (default: 500Mi)
    driver_mem_overhead: <string>  # off-heap (non-JVM) memory allocated to the driver (overrides mem_overhead_factor) (default: min[driver_mem * 0.4, 384Mi])
    executor_cpu: <string>  # CPU request for each spark executor (default: 1)
    executor_mem: <string>  # memory request for each spark executor (default: 500Mi)
    executor_mem_overhead: <string>  # off-heap (non-JVM) memory allocated to each executor (overrides mem_overhead_factor) (default: min[executor_mem * 0.4, 384Mi])
    mem_overhead_factor: <float>  # the proportion of driver_mem/executor_mem which will be additionally allocated for off-heap (non-JVM) memory (default: 0.4)
```

See [Data Types](data-types.md) for details about input values. Note: the `input` of the the aggregate must match the input type of the aggregator (if specified).

See <!-- CORTEX_VERSION_MINOR -->[`aggregators.yaml`](https://github.com/cortexlabs/cortex/blob/master/pkg/aggregators/aggregators.yaml) for a list of built-in aggregators.

## Example

```yaml
- kind: aggregate
  name: age_bucket_boundaries
  aggregator: cortex.bucket_boundaries
  input:
    col: @age  # "age" is the name of a numeric raw column
    num_buckets: 5

- kind: aggregate
  name: price_bucket_boundaries
  aggregator: cortex.bucket_boundaries
  input:
    col: @price  # "price" is the name of a numeric raw column
    num_buckets: @num_buckets  # "num_buckets" is the name of an INT constant
```
