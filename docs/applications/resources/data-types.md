# Data Types

Data types are used in config files to help validate data and ensure your Cortex application is functioning as expected.

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

## Input Column Types

Some resources specify the types of columns that are to be used as inputs (e.g. `transformer.inputs.columns` and `aggregator.inputs.columns`). For these types, any of the column types may be used:

* `INT_COLUMN`
* `FLOAT_COLUMN`
* `STRING_COLUMN`
* `INT_LIST_COLUMN`
* `FLOAT_LIST_COLUMN`
* `STRING_LIST_COLUMN`

Ambiguous input types are also supported, and are represented by joining column types with `|`. For example, `INT_COLUMN|FLOAT_COLUMN` indicates that either a column of type `INT_COLUMN` or a column of type `FLOAT_COLUMN` may be used an the input. Any two or more column types may be combined in this way (e.g. `INT_COLUMN|FLOAT_COLUMN|STRING_COLUMN` is supported). All permutations of ambiguous types are valid (e.g. `INT_COLUMN|FLOAT_COLUMN` and `FLOAT_COLUMN|INT_COLUMN` are equivalent).

In addition, an input type may be a list of columns. To denote this, use any of the supported input column types in a length-one list. For example, `[INT_COLUMN]` represents a list of integer columns; `[INT_COLUMN|FLOAT_COLUMN]` represents a list of integer or float columns.

Note: `[INT_COLUMN]` is not equivalent to `INT_LIST_COLUMN`: the former denotes a list of integer columns, whereas the latter denotes a single column which contains a list of integers.

## Value types

These are valid types for all values (e.g. aggregator args, aggregator output types, transformer args, constants).

* `INT`
* `FLOAT`
* `STRING`
* `BOOL`

As with column input types, ambiguous types are supported (e.g. `INT|FLOAT`), length-one lists of types are valid (e.g. `[STRING]`), and all permutations of ambiguous types are valid (e.g. `INT|FLOAT` is equivalent to `FLOAT|INT`).

In addition, maps are valid value types. There are two types of maps: maps with a single data type key, and maps with any number of arbitrary keys:

### Maps with a single data type key

This represents a map which, at runtime, may have any number of items. The types of the keys and values must match the declared types.

Example: `{STRING: INT}`

Example value: `{"San Francisco": -7, "Toronto": -4}`

### Maps with arbitrary keys

This represents a map which, at runtime, must define values for each of the pre-defined keys.

Example: `{"value1": INT, "value2": FLOAT}`

Example value: `{"value1": 17, "value2": 8.8}`

### Nested maps

The values in either of the map types may be arbitrarily nested data types.

## Examples

* `INT`
* `[FLOAT|STRING]`
* `{STRING: INT}`
* `{"value1": INT, "value2": FLOAT}`
* `{"value1": {STRING: BOOL|INT}, "value2": [FLOAT], "value2": STRING}`
