# Constants

Constants represent literal values which can be used in other Cortex configuration files. They can be useful for extracting repetitive literals into a single variable.

## Config

```yaml
- kind: constant  # (required)
  name: <string>  # constant name (required)
  type: <value_type>  # the data type of the constant (required)
  value: <value>  # a literal value (required)
  tags:
    <string>: <scalar>  # arbitrary key/value pairs to attach to the resource (optional)
    ...
```

## Example

```yaml
- kind: constant
  name: num_buckets
  type: INT
  value: 5

- kind: constant
  name: bucket_boundaries
  type: [FLOAT]
  value: [0, 50, 100]
```
