# Templates

Templates allow you to reuse resource configuration within your application.

## Config

```yaml
- kind: template  # (required)
  name: <string>  # template name (required)
  yaml: <string>  # YAML string including named arguments in {} (required)

- kind: embed  # (required)
  template: <string>  # reference to a Cortex template (required)
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
      name: {feature}_mean
      aggregator: cortex.mean
      inputs:
        features:
          col: {feature}

    - kind: aggregate
      name: {feature}_stddev
      aggregator: cortex.stddev
      inputs:
        features:
          col: {feature}

    - kind: transformed_feature
      name: {feature}_normalized
      tags:
        type: numeric
      transformer: cortex.normalize
      inputs:
        features:
          num: {feature}
        args:
          mean: {feature}_mean
          stddev: {feature}_stddev

- kind: embed
  template: normalize
  args:
    feature: feature1

- kind: embed
  template: normalize
  args:
    feature: feature2
```
