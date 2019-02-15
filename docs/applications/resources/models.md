# Models

Train custom TensorFlow models at scale.

## Config

```yaml
- kind: <string>  # (required)
  name: <string>  # model name (required)
  type: <string>  # "classification" or "regression" (required)
  target_column: <string>  # the column to predict (must be an integer column for classification, or an integer or float column for regression) (required)
  feature_columns: <[string]>  # a list of the columns used as input for this model (required)
  training_columns: <[string]>  # a list of the columns used only during training (optional)
  aggregates: <[string]>  # a list of aggregates to pass into model training (optional)
  hparams: <map>  # a map of hyperparameters to pass into model training (optional)
  prediction_key: <string>  # key of the target value in the estimator's exported predict outputs (default: "class_ids" for classification, "predictions" for regression)
  path: <string>  # path to the implementation file, relative to the application root (default: implementations/models/<name>.py)

  data_partition_ratio:
    training: <float>  # the proportion of data to be used for training (default: 0.8)
    evaluation: <float>  # the proportion of data to be used for evaluation (default: 0.2)

  training:
    batch_size: <int>  # training batch size (default: 40)
    num_steps: <int>  # number of training steps (default: 1000)
    num_epochs: <int>  # number of epochs to train the model over the entire dataset (optional)
    shuffle: <boolean>  # whether to shuffle the training data (default: true)
    tf_random_seed: <int>  # random seed for TensorFlow initializers (default: <random>)
    save_summary_steps: <int>  # save summaries every this many steps (default: 100)
    log_step_count_steps: <int>  # the frequency, in number of global steps, that the global step/sec and the loss will be logged during training (default: 100)
    save_checkpoints_secs: <int>  # save checkpoints every this many seconds (default: 600)
    save_checkpoints_steps: <int>  # save checkpoints every this many steps (default: 100)
    keep_checkpoint_max: <int>  # the maximum number of recent checkpoint files to keep (default: 3)
    keep_checkpoint_every_n_hours: <int>  # number of hours between each checkpoint to be saved (default: 10000)

  evaluation:
    batch_size: <int>  # evaluation batch size (default: 40)
    num_steps: <int>  # number of eval steps (default: 100)
    num_epochs: <int>  # number of epochs to evaluate the model over the entire dataset (optional)
    shuffle: <bool>  # whether to shuffle the evaluation data (default: false)
    start_delay_secs: <int>  # start evaluating after waiting for this many seconds (default: 120)
    throttle_secs: <int>  # do not re-evaluate unless the last evaluation was started at least this many seconds ago (default: 600)

  compute:
    cpu: <string>  # CPU request (default: Null)
    mem: <string>  # memory request (default: Null)
    gpu: <string>  # GPU request (default: Null)

  tags:
    <string>: <scalar>  # arbitrary key/value pairs to attach to the resource (optional)
    ...
```

See the [tf.estimator.RunConfig](https://www.tensorflow.org/api_docs/python/tf/estimator/RunConfig) and [tf.estimator.EvalSpec](https://www.tensorflow.org/api_docs/python/tf/estimator/EvalSpec) documentation for more information.

## Example

```yaml
- kind: model
  name: dnn
  type: classification
  target_column: label
  feature_columns:
    - column1
    - column2
    - column3
  training_columns:
    - class_weight
  aggregates:
    - column1_index
  hparams:
    hidden_units: [4, 2]
  data_partition_ratio:
    training: 0.9
    evaluation: 0.1
  training:
    batch_size: 10
    num_steps: 1000
```
