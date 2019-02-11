# Templates

Templates allow you to reuse resource configuration within your application.

## Config

```yaml
- kind: template  # (required)
  name: <string>  # template name (required)
  yaml: <string>  # YAML string including named arguments enclosed by {} (required)

- kind: embed  # (required)
  template: <string>  # name of a Cortex template (required)
  args:
    <string>: <value>  # (required)
    ...
```

## Example

```yaml
- kind: template
  name: normalize
  yaml: |
    - kind: aggregate
      name: {column}_mean
      aggregator: cortex.mean
      inputs:
        columns:
          col: {column}

    - kind: aggregate
      name: {column}_stddev
      aggregator: cortex.stddev
      inputs:
        columns:
          col: {column}

    - kind: transformed_column
      name: {column}_normalized
      tags:
        type: numeric
      transformer: cortex.normalize
      inputs:
        columns:
          num: {column}
        args:
          mean: {column}_mean
          stddev: {column}_stddev

- kind: embed
  template: normalize
  args:
    column: column1

- kind: embed
  template: normalize
  args:
    column: column2
```
