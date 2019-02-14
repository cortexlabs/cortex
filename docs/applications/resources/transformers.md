# Transformers

A transformer converts a set of columns and arbitrary values into a single transformed column. Each transformer has an input schema and an output data type. The input schema is a map which specifies the name and data type of each input column and argument.

## Config

```yaml
- kind: transformer
  name: <string>  # transformer name (required)
  path: <string>  # path to the implementation file, relative to the application root (default: implementations/transformers/<name>.py)
  output_type: <transformed_column_type>  # output data type (required)
  inputs:
    columns:
      <string>: <input_column_type>  # map of column input name to column input type(s) (required)
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
  output_type: FLOAT_COLUMN
  inputs:
    columns:
      num: INT_COLUMN|FLOAT_COLUMN
    args:
      mean: FLOAT
      stddev: FLOAT
```

## Built-in Transformers

Cortex includes common transformers that can be used out of the box (see <!-- CORTEX_VERSION_MINOR -->[`transformers.yaml`](https://github.com/cortexlabs/cortex/blob/0.1/pkg/transformers/transformers.yaml)). To use built-in transformers, use the `cortex` namespace in the transformer name (e.g. `cortex.normalize`).
