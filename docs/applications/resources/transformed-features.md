# Transformed Features

Transform feature data at scale.

## Config

```yaml
- kind: transformed_feature
  name: <string>  # transformed feature name (required)
  transformer: <string>  # the name of the transformer to use (required)
  inputs:
    features:
      <string>: <string> or <[string]>  # map of feature input name to raw feature name(s) (required)
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

Note: the `features` and `args` fields of the the transformed feature must match the data types of the `features` and `args` fields of the selected transformer.

Each `args` value may be the name of an aggregate, the name of a constant, or a literal value. Any string value will be assumed to be the name of an aggregate or constant. To use a string literal as an arg, escape it with double quotes (e.g. `arg_name: "\"string literal\""`.

See our [`transformers.yaml`](../../../pkg/transformers/transformers.yaml) file for a list of built-in transformers.

## Example

```yaml
- kind: transformed_feature
  name: age_normalized
  transformer: cortex.normalize
  inputs:
    features:
      num: age  # feature name
    args:
      mean: age_mean  # the name of a cortex.mean aggregator
      stddev: age_stddev  # the name of a cortex.stddev aggregator

- kind: transformed_feature
  name: class_indexed
  transformer: cortex.index_string
  inputs:
    features:
      col: class  # the name of a string feature
    args:
      index: ["t", "f"]  # a value to be used as the index

- kind: transformed_feature
  name: price_bucketized
  transformer: cortex.bucketize
  inputs:
    features:
      num: price  # feature name
    args:
      bucket_boundaries: bucket_boundaries  # the name of a [FLOAT] constant
```

## Validating Transformers

Cortex does not run feature transformers on the full dataset until they are required for model training. However, in order to catch bugs as early as possible, Cortex sanity checks all transformed features by running their transformers against the first 100 samples in the dataset.
