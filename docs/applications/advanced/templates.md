# Templates

Templates allow you to reuse resource configuration within your application.

## Config

```yaml
- kind: template
  name: <string>  # template name (required)
  yaml: <string>  # YAML string including named arguments enclosed by {} (required)

- kind: embed
  template: <string>  # name of a Cortex template (required)
  args:
    <string>: <value>  # (required)
```

## Example

```yaml
- kind: template
  name: normalize
  yaml: |
    - kind: aggregate
      name: {column}_mean
      aggregator: cortex.mean
      input: @{column}

    - kind: aggregate
      name: {column}_stddev
      aggregator: cortex.stddev
      input: @{column}

    - kind: transformed_column
      name: {column}_normalized
      transformer: cortex.normalize
      input:
        col: @{column}
        mean: @{column}_mean
        stddev: @{column}_stddev

- kind: embed
  template: normalize
  args:
    column: column1

- kind: embed
  template: normalize
  args:
    column: column2
```
