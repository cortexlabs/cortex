# Data Types

Data types are used in config files to help validate data and ensure your Cortex application is functioning as expected.

## Raw Feature Types

These are the valid types for raw features:

* `INT_FEATURE`
* `FLOAT_FEATURE`
* `STRING_FEATURE`

## Transformed Feature Types

These are the valid types for transformed features (i.e. output types of transformers):

* `INT_FEATURE`
* `FLOAT_FEATURE`
* `STRING_FEATURE`
* `INT_LIST_FEATURE`
* `FLOAT_LIST_FEATURE`
* `STRING_LIST_FEATURE`

## Input Feature Types

Some resources specify the types of features that are to be used as inputs (e.g. `transformer.inputs.features` and `aggregator.inputs.features`). For these types, any of the feature types may be used:

* `INT_FEATURE`
* `FLOAT_FEATURE`
* `STRING_FEATURE`
* `INT_LIST_FEATURE`
* `FLOAT_LIST_FEATURE`
* `STRING_LIST_FEATURE`

Ambiguous input types are also supported, and are represented by joining feature types with `|`. For example, `INT_FEATURE|FLOAT_FEATURE` indicates that either a feature of type `INT_FEATURE` or a feature of type `FLOAT_FEATURE` may be used an the input. Any two or more feature types may be combined in this way (e.g. `INT_FEATURE|FLOAT_FEATURE|STRING_FEATURE` is supported). All permutations of ambiguous types are valid (e.g. `INT_FEATURE|FLOAT_FEATURE` and `FLOAT_FEATURE|INT_FEATURE` are equivalent).

In addition, an input type may be a list of features. To denote this, use any of these types in a length-one list. For example, `[INT_FEATURE]` represents a list of integer features; `[INT_FEATURE|FLOAT_FEATURE]` represents a list of integer or float features.

Note: `[INT_FEATURE]` is not equivalent to `INT_LIST_FEATURE`: the former denotes a list of integer features, whereas the latter denotes a single feature which contains a list of integers.

## Value types

These are valid types for all values (aggregator args, aggregator output types, transformer args, and constants).

* `INT`
* `FLOAT`
* `STRING`
* `BOOL`

As with feature input types, ambiguous types are supported (e.g. `INT|FLOAT`), length-one lists of types are valid (e.g. `[STRING]`), and all permutations of ambiguous types are valid (e.g. `INT|FLOAT` is equivalent to `FLOAT|INT`).

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
