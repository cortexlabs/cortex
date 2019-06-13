# Data Types

Data types are used in configuration files to help validate data and ensure your Cortex application is functioning as expected.


## Raw Column Types

These are the valid types for raw columns:

* `INT_COLUMN`
* `FLOAT_COLUMN`
* `STRING_COLUMN`


## Transformed Column Types

These are the valid types for transformed columns (i.e. output types of transformers):

* `INT_COLUMN`
* `FLOAT_COLUMN`
* `STRING_COLUMN`
* `INT_LIST_COLUMN`
* `FLOAT_LIST_COLUMN`
* `STRING_LIST_COLUMN`


## Output Types

Output types are used to define the output types of aggregators and constants. There are four base scalar types:

* `INT`
* `FLOAT`
* `STRING`
* `BOOL`

In addition, an output type may be a list of scalar types. This is denoted by a length-one list of any of the supported types. For example, `[INT]` represents a list of integers.

An output type may also be a map containing these types. There are two types of maps: generic maps and fixed maps. **Generic maps** represent maps which may have any number of items. The types of the keys and values must match the declared types. For example, `{STRING: INT}` supports `{"San Francisco": -7, "Toronto": -4}`. **Fixed maps** represent maps which must define values for each of the pre-defined keys. For example: `{mean: INT, stddev: FLOAT}` supports `{mean: 17, stddev: 8.8}`.

The values in lists, generic maps, and fixed maps may be arbitrarily nested.

Here are some valid output types:

* `INT`
* `[STRING]`
* `{STRING: INT}`
* `[{STRING: INT}]`
* `{value1: INT, value2: FLOAT}`
* `{STRING: {value1: INT, value2: FLOAT}}`
* `{value1: {STRING: BOOL}, value2: [FLOAT], value2: STRING}`

### Example

Output type:

```yaml
output_type:
  value1: BOOL
  value2: INT|FLOAT
  value3: [STRING]
  value4: {INT: STRING}
```

Output value:

```yaml
output_type:
  value1: True
  value2: 2.2
  value3: [test1, test2, test3]
  value4: {1: test1, 2: test2}
```


## Input Types

Input types are used to define the inputs to aggregators, transformers, and estimators. Typically, input types can be any combination of the column or scalar types:

* `INT_COLUMN`
* `FLOAT_COLUMN`
* `STRING_COLUMN`
* `INT_LIST_COLUMN`
* `FLOAT_LIST_COLUMN`
* `STRING_LIST_COLUMN`
* `INT`
* `FLOAT`
* `STRING`
* `BOOL`

Like with output types, input types may occur within arbitrarily nested lists, generic maps, and fixed maps.

Ambiguous input types are also supported, and are represented by joining types with `|`. For example, `INT_COLUMN|FLOAT_COLUMN` indicates that either a column of type `INT_COLUMN` or a column of type `FLOAT_COLUMN` may be used as the input. Any two or more types may be combined in this way (e.g. `INT|FLOAT|STRING` is supported). All permutations of ambiguous types are valid (e.g. `INT|FLOAT` and `FLOAT|INT` are equivalent). Column types and scalar types may not be combined (e.g. `INT|FLOAT_COLUMN` is not valid).

### Input Type Validations

By default, all declared inputs are required. For example, if the input type is `{value1: INT, value2: FLOAT}`, both `value1` and `value2` must be provided (and cannot be `Null`). With Cortex, it is possible to declare inputs as optional, set default values, allow values to be `Null`, and specify minimum and maximum map/list lengths.

To specify validation options, the "long form" input schema is used. In the long form, the input type is always a map, with the `_type` key specifying the type, and other keys (which all start with `_`) specifying the options. The available options are:

* `_optional`: If set to `True`, allows the value to be missing from the input. This only applies to values in maps.
* `_default`: Specifies a default value to use if the value is missing from the input. This only applies to values in maps. Setting `_defaut` implies `_optional: True`.
* `_allow_null`: If set to `True`, allows the value to be explicitly set to `Null`.
* `_min_count`: Specifies the minimum number of elements that must be in the list or map.
* `_max_count`: Specifies the maximum number of elements that must be in the list or map.

### Example

Short form input type:

```yaml
input:
  value1: INT_COLUMN
  value2: INT|FLOAT
  value3: [STRING]
  value4: {INT: STRING}
```

Long form input type:

```yaml
input:
  value1:
    _type: INT_COLUMN
    _optional: True
  value2:
    _type: INT|FLOAT
    _default: 2.2
    _allow_null: True
  value3:
    _type: [STRING]
    _min_count: 1
  value4:
    _type: {INT: STRING}
    _min_count: 1
    _max_count: 100
```

Input value (assuming `column1` is an `INT_COLUMN`, `constant1` is a `[STRING]`, and `aggregate1` is an `STRING`):

```yaml
input:
  value1: @column1
  value2: 2.2
  value3: @constant1
  value4: {1: test1, 2: @aggregate1}
```
