# Constants

Constants represent literal values which can be used in other Cortex resources. They can be useful for extracting repetitive literals into a single variable.

## Config

```yaml
- kind: constant
  name: <string>  # constant name (required)
  type: <output_type>  # the type of the constant (optional, will be inferred from value if not specified)
  value: <output_value>  # a literal value (this or path must be specified)
  path: <string>  # path to a JSON object (e.g. s3://my-bucket/constant.json) (this or value must be specified)

```

See [Data Types](data-types.md) for details about output types and values.

## Example

```yaml
- kind: constant
  name: num_buckets
  value: 5

- kind: constant
  name: bucket_boundaries
  type: [FLOAT]
  value: [0, 50, 100]
```
