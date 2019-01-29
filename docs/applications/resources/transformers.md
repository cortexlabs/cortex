# Transformers

A transformer converts a set of features and arbitrary arguments into a single transformed feature. Each transformer has an input schema and an output data type. The input schema is a map which specifies the name and data type of each input feature and argument.

## Config

```yaml
- kind: transformer
  name: <string>  # transformer name (required)
  path: <string>  # path to the implementation file, relative to the application root (default: implementations/transformers/<name>.py)
  output_type: <transformed_feature_type>  # output data type (required)
  inputs:
    features:
      <string>: <input_feature_type>  # map of feature input name to feature input type(s) (required)
      ...
    args:
      <string>: <value_type>  # map of arg input name to value input type(s) (optional)
      ...
```

See [Data Types](datatypes.md) for a list of valid data types.

## Example

```yaml
- kind: transformer
  name: normalize
  output_type: FLOAT_FEATURE
  inputs:
    features:
      num: INT_FEATURE|FLOAT_FEATURE
    args:
      mean: FLOAT
      stddev: FLOAT
```

## Built-in Transformers

[comment]: <> (CORTEX_VERSION_MINOR)

Cortex includes common transformers that can be used out of the box (see [`transformers.yaml`](https://github.com/cortexlabs/cortex/blob/master/pkg/transformers/transformers.yaml)). To use built-in transformers, use the `cortex` namespace in the transformer name (e.g. `cortex.normalize`).
