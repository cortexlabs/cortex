# Transformed Columns

Transform data at scale.

## Config

```yaml
- kind: transformed_column
  name: <string>  # transformed column name (required)
  transformer: <string>  # the name of the transformer to use (this or transformer_path must be specified)
  transformer_path: <string>  # a path to an transformer implementation file (this or transformer must be specified)
  input: <input_value>  # the input to the transformer, which may contain references to columns, constants, and aggregates (e.g. @column1) (required)
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

See [Data Types](data-types.md) for details about input values. Note: the `input` of the the transformed column must match the input type of the transformer (if specified).

See <!-- CORTEX_VERSION_MINOR -->[`transformers.yaml`](https://github.com/cortexlabs/cortex/blob/0.6/pkg/transformers/transformers.yaml) for a list of built-in transformers.

## Example

```yaml
- kind: transformed_column
  name: age_normalized
  transformer: cortex.normalize
  input:
    num: @age  # "age" is the name of a numeric raw column
    mean: @age_mean  # "age_mean" is the name of an aggregate which used the cortex.mean aggregator
    stddev: @age_stddev  # "age_stddev" is the name of an aggregate which used the cortex.stddev aggregator

- kind: transformed_column
  name: class_indexed
  transformer: cortex.index_string
  input:
    col: @class  # "class" is the name of a string raw column
    index: {"indexes": ["t", "f"], "reversed_index": ["t": 0, "f": 1]}  # a value to be used as the index

- kind: transformed_column
  name: price_bucketized
  transformer: cortex.bucketize
  input:
    num: @price  # "price" is the name of a numeric raw column
    bucket_boundaries: @bucket_boundaries  # "bucket_boundaries" is the name of a [FLOAT] constant
```

## Validating Transformers

Cortex does not run transformers on the full dataset until they are required for model training. However, in order to catch bugs as early as possible, Cortex sanity checks all transformed columns by running their transformers against the first 100 samples in the dataset.
