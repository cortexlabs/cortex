# Copyright 2019 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import sys
import inspect
import importlib
import multiprocessing
import math
import tensorflow as tf

from lib import util, tf_lib
from lib.exceptions import UserRuntimeException


def get_input_placeholder(model_name, ctx, training=True):
    column_types = tf_lib.get_column_tf_types(model_name, ctx, training)
    input_placeholder = {
        feature_name: tf.placeholder(shape=[None], dtype=tf_type)
        for feature_name, tf_type in column_types.items()
    }
    return input_placeholder


def get_label_placeholder(model_name, ctx):
    model = ctx.models[model_name]

    target_column_name = model["target_column"]
    column_type = tf_lib.CORTEX_TYPE_TO_TF_TYPE[ctx.columns[target_column_name]["type"]]
    return tf.placeholder(shape=[None], dtype=column_type)


def get_transform_tensor_fn(ctx, model_impl, model_name):
    model = ctx.models[model_name]
    model_config = ctx.model_config(model["name"])

    def transform_tensor_fn_wrapper(inputs, labels):
        return model_impl.transform_tensorflow(inputs, labels, model_config)

    return transform_tensor_fn_wrapper


def generate_example_parsing_fn(model_name, ctx, training=True):
    model = ctx.models[model_name]

    feature_spec = tf_lib.get_feature_spec(model_name, ctx, training)

    def _parse_example(example_proto):
        features = tf.parse_single_example(serialized=example_proto, features=feature_spec)
        target = features.pop(model["target_column"], None)
        return features, target

    return _parse_example


# Mode must be "training" or "evaluation"
def generate_input_fn(model_name, ctx, mode, model_impl):
    model = ctx.models[model_name]

    filenames = ctx.get_training_data_parts(model_name, mode)
    filenames = [ctx.storage.blob_path(f) for f in filenames]

    num_threads = multiprocessing.cpu_count()
    buffer_size = 2 * model[mode]["batch_size"] + 1

    def _input_fn():
        dataset = tf.data.TFRecordDataset(filenames=filenames)
        dataset = dataset.map(
            generate_example_parsing_fn(model_name, ctx, training=True),
            num_parallel_calls=num_threads,
        )

        if model[mode]["shuffle"]:
            dataset = dataset.shuffle(buffer_size)

        if hasattr(model_impl, "transform_tensorflow"):
            dataset = dataset.map(get_transform_tensor_fn(ctx, model_impl, model_name))

        dataset = dataset.batch(model[mode]["batch_size"])
        dataset = dataset.prefetch(buffer_size)
        dataset = dataset.repeat()
        iterator = dataset.make_one_shot_iterator()
        features, target = iterator.get_next()

        return features, target

    return _input_fn


def generate_json_serving_input_fn(model_name, ctx, model_impl):
    def _json_serving_input_fn():
        inputs = get_input_placeholder(model_name, ctx, training=False)
        labels = get_label_placeholder(model_name, ctx)

        features = {key: tensor for key, tensor in inputs.items()}
        if hasattr(model_impl, "transform_tensorflow"):
            features, _ = get_transform_tensor_fn(ctx, model_impl, model_name)(features, labels)

        features = {key: tf.expand_dims(tensor, 0) for key, tensor in features.items()}
        return tf.estimator.export.ServingInputReceiver(features=features, receiver_tensors=inputs)

    return _json_serving_input_fn


def get_regression_eval_metrics(labels, predictions):
    metrics = {}
    prediction_values = predictions["predictions"]
    metrics["RMSE"] = tf.metrics.root_mean_squared_error(
        labels=labels, predictions=prediction_values
    )
    metrics["MAE"] = tf.metrics.mean_absolute_error(labels=labels, predictions=prediction_values)

    return metrics


def train(model_name, model_impl, ctx, model_dir):
    model = ctx.models[model_name]

    util.mkdir_p(model_dir)
    util.rm_dir(model_dir)

    tf_lib.set_logging_verbosity(ctx.environment["log_level"]["tensorflow"])

    run_config = tf.estimator.RunConfig(
        tf_random_seed=model["training"]["tf_random_seed"],
        save_summary_steps=model["training"]["save_summary_steps"],
        save_checkpoints_secs=model["training"]["save_checkpoints_secs"],
        save_checkpoints_steps=model["training"]["save_checkpoints_steps"],
        log_step_count_steps=model["training"]["log_step_count_steps"],
        keep_checkpoint_max=model["training"]["keep_checkpoint_max"],
        keep_checkpoint_every_n_hours=model["training"]["keep_checkpoint_every_n_hours"],
        model_dir=model_dir,
    )

    train_input_fn = generate_input_fn(model_name, ctx, "training", model_impl)
    eval_input_fn = generate_input_fn(model_name, ctx, "evaluation", model_impl)
    serving_input_fn = generate_json_serving_input_fn(model_name, ctx, model_impl)
    exporter = tf.estimator.FinalExporter("estimator", serving_input_fn, as_text=False)

    train_num_steps = model["training"]["num_steps"]
    dataset_metadata = ctx.get_metadata("training_datasets", model_name)
    if model["training"]["num_epochs"]:
        train_num_steps = (
            math.ceil(dataset_metadata["training_size"] / float(model["training"]["batch_size"]))
            * model["training"]["num_epochs"]
        )

    train_spec = tf.estimator.TrainSpec(train_input_fn, max_steps=train_num_steps)

    eval_num_steps = model["evaluation"]["num_steps"]
    if model["evaluation"]["num_epochs"]:
        eval_num_steps = (
            math.ceil(dataset_metadata["eval_size"] / float(model["evaluation"]["batch_size"]))
            * model["evaluation"]["num_epochs"]
        )

    eval_spec = tf.estimator.EvalSpec(
        eval_input_fn,
        steps=eval_num_steps,
        exporters=[exporter],
        name="estimator-eval",
        start_delay_secs=model["evaluation"]["start_delay_secs"],
        throttle_secs=model["evaluation"]["throttle_secs"],
    )

    model_config = ctx.model_config(model["name"])
    tf_lib.add_tf_types(model_config)

    try:
        estimator = model_impl.create_estimator(run_config, model_config)
    except Exception as e:
        raise UserRuntimeException("model " + model_name) from e

    if model["type"] == "regression":
        estimator = tf.contrib.estimator.add_metrics(estimator, get_regression_eval_metrics)

    tf.estimator.train_and_evaluate(estimator, train_spec, eval_spec)

    return model_dir
