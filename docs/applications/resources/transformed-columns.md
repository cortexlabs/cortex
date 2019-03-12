# Transformed Columns

Transform data at scale.

## Config

```yaml
- kind: transformed_column
  name: <string>  # transformed column name (required)
  transformer: <string>  # the name of the transformer to use (required)
  inputs:
    columns:
      <string>: <string> or <[string]>  # map of column input name to raw column name(s) (required)
      ...
    args:
      <string>: <value>  # value may be an aggregate, constant, or literal value (optional)
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

Note: the `columns` and `args` fields of the the transformed column must match the data types of the `columns` and `args` fields of the selected transformer.

Each `args` value may be the name of an aggregate, the name of a constant, or a literal value. Any string value will be assumed to be the name of an aggregate or constant. To use a string literal as an arg, escape it with double quotes (e.g. `arg_name: "\"string literal\""`.

See <!-- CORTEX_VERSION_MINOR -->[`transformers.yaml`](https://github.com/cortexlabs/cortex/blob/0.2/pkg/transformers/transformers.yaml) for a list of built-in transformers.

## Example

```yaml
- kind: transformed_column
  name: age_normalized
  transformer: cortex.normalize
  inputs:
    columns:
      num: age  # column name
    args:
      mean: age_mean  # the name of a cortex.mean aggregator
      stddev: age_stddev  # the name of a cortex.stddev aggregator

- kind: transformed_column
  name: class_indexed
  transformer: cortex.index_string
  inputs:
    columns:
      col: class  # the name of a string column
    args:
      index: ["t", "f"]  # a value to be used as the index

- kind: transformed_column
  name: price_bucketized
  transformer: cortex.bucketize
  inputs:
    columns:
      num: price  # column name
    args:
      bucket_boundaries: bucket_boundaries  # the name of a [FLOAT] constant
```

## Validating Transformers

Cortex does not run transformers on the full dataset until they are required for model training. However, in order to catch bugs as early as possible, Cortex sanity checks all transformed columns by running their transformers against the first 100 samples in the dataset.
