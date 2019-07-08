# Estimators

An estimator defines how to train a model.

Custom estimators can be implemented in Python or PySpark. See the [implementation docs](estimators.md) for a detailed guide.

## Config

```yaml
- kind: estimator
  name: <string>  # estimator name (required)
  path: <string>  # path to the implementation file, relative to the cortex root (default: implementations/estimators/<name>.py)
  target_column: <column_type>  # The type of column that can be used as a target (ambiguous types like INT_COLUMN|FLOAT_COLUMN are supported) (required)
  input: <input_type>  # the input type of the estimator (required)
  training_input: <input_type>  # the input type of the training input to the estimator (optional)
  hparams: <input_type>  # the input type of the hyperparameters to pass into the estimator, which may not contain column types (optional)
  prediction_key: <string>  # key of the target value in the estimator's exported predict outputs (default: "class_ids" for INT_COLUMN and STRING_COLUMN targets, "predictions" otherwise)
```

See [Data Types](data-types.md) for details about input types.

## Example

```yaml
- kind: estimator
  name: dnn_classifier
  path: dnn_classifier.py
  target_column: INT_COLUMN
  input:
    num_classes: INT
    numeric_columns: [INT_COLUMN|FLOAT_COLUMN]
  hparams:
    hidden_units: [INT]
```

## Built-in Estimators

Cortex includes common estimators that can be used out of the box (see <!-- CORTEX_VERSION_MINOR -->[`estimators.yaml`](https://github.com/cortexlabs/cortex/blob/0.6/pkg/estimators/estimators.yaml)). To use built-in estimators, use the `cortex` namespace in the estimator name (e.g. `cortex.dnn_classifier`).
