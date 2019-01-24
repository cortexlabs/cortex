# Raw Features

Validate raw data at scale and define features.

## Config

```yaml
- kind: raw_feature
  name: <string>  # raw feature name (required)
  type: INT_FEATURE  # data type (required)
  required: <boolean>  # whether null values are allowed (default: false)
  min: <int>  # minimum allowed value (optional)
  max: <int>  # maximum allowed value (optional)
  values: <[int]>  # an exhaustive list of allowed values (optional)
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

- kind: raw_feature
  name: <string>  # raw feature name (required)
  type: FLOAT_FEATURE  # data type (required)
  required: <boolean>  # whether null values are allowed (default: false)
  min: <float>  # minimum allowed value (optional)
  max: <float>  # maximum allowed value (optional)
  values: <[float]>  # an exhaustive list of allowed values (optional)
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

- kind: raw_feature
  name: <string>  # raw feature name (required)
  type: STRING_FEATURE  # data type (required)
  required: <boolean>  # whether null values are allowed (default: false)
  values: <[string]>  # an exhaustive list of allowed values (optional)
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

## Example

```yaml
- kind: raw_feature
  name: feature1
  type: INT_FEATURE
  required: true
  min: 0
  max: 10

- kind: raw_feature
  name: feature2
  type: FLOAT_FEATURE
  required: true
  min: 1.1
  max: 2.2

- kind: raw_feature
  name: feature3
  type: STRING_FEATURE
  required: false
  values: [a, b, c]
```

## Data Validation

Cortex integrates with your existing data warehouse and runs all validations every time new data is ingested. All raw features are cached to speed up additional processing.
