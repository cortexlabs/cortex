# Models

Train TensorFlow models at scale.

## Config

```yaml
- kind: model
  name: <string>  # model name (required)
  estimator: <string>  # the name of the estimator to use (this or estimator_path must be specified)
  estimator_path: <string>  # a path to an estimator implementation file (this or estimator must be specified)
  target_column: <column_reference>  # a reference to the column to predict (e.g. @column1) (required)
  input: <input_value>  # the input to the model, which may contain references to columns, constants, and aggregates (e.g. @column1) (required)
  training_input: <input_value> # input to the model which is only used during training, which may contain references to columns, constants, and aggregates (e.g. @column1) (optional)
  hparams: <output_value>  # hyperparameters to pass into model training, which may not contain reference to other resources (optional)
  prediction_key: <string>  # key of the target value in the estimator's exported predict outputs (default: "class_ids" for INT_COLUMN and STRING_COLUMN targets, "predictions" otherwise)

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

  compute:         # Resources for training and evaluations steps (TensorFlow)
    cpu: <string>  # CPU request (default: Null)
    mem: <string>  # memory request (default: Null)
    gpu: <string>  # GPU request (default: Null)

  dataset_compute:    # Resources for constructing training dataset (Spark)
    executors: <int>  # number of spark executors (default: 1)
    driver_cpu: <string>  # CPU request for spark driver (default: 1)
    driver_mem: <string>  # memory request for spark driver (default: 500Mi)
    driver_mem_overhead: <string>  # off-heap (non-JVM) memory allocated to the driver (overrides mem_overhead_factor) (default: min[driver_mem * 0.4, 384Mi])
    executor_cpu: <string>  # CPU request for each spark executor (default: 1)
    executor_mem: <string>  # memory request for each spark executor (default: 500Mi)
    executor_mem_overhead: <string>  # off-heap (non-JVM) memory allocated to each executor (overrides mem_overhead_factor) (default: min[executor_mem * 0.4, 384Mi])
    mem_overhead_factor: <float>  # the proportion of driver_mem/executor_mem which will be additionally allocated for off-heap (non-JVM) memory (default: 0.4)
```

See [Data Types](data-types.md) for details about input and output values. Note: the `target_column`, `input`, `training_input`, and `hparams` of the the aggregate must match the input types of the estimator (if specified).

See the [tf.estimator.RunConfig](https://www.tensorflow.org/api_docs/python/tf/estimator/RunConfig) and [tf.estimator.EvalSpec](https://www.tensorflow.org/api_docs/python/tf/estimator/EvalSpec) documentation for more information.

## Example

```yaml
- kind: model
  name: dnn
  estimator: cortex.dnn_classifier
  target_column: @class
  input:
    numeric_columns: [@column1, @column2, @column3]
  hparams:
    hidden_units: [4, 2]
  data_partition_ratio:
    training: 0.9
    evaluation: 0.1
  training:
    batch_size: 10
    num_steps: 1000
```
