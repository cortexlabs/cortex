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

from lib import util, tf_lib, aws
from lib.exceptions import UserRuntimeException


def get_input_placeholder(model_name, ctx, training=True):
    feature_types = tf_lib.get_feature_tf_types(model_name, ctx, training)
    input_placeholder = {
        feature_name: tf.placeholder(shape=[None], dtype=tf_type)
        for feature_name, tf_type in feature_types.items()
    }
    return input_placeholder


def generate_example_parsing_fn(model_name, ctx, training=True):
    model = ctx.models[model_name]

    feature_spec = tf_lib.get_feature_spec(model_name, ctx, training)

    def _parse_example(example_proto):
        features = tf.parse_single_example(serialized=example_proto, features=feature_spec)
        target = features.pop(model["target"], None)
        return features, target

    return _parse_example


# Mode must be "training" or "evaluation"
def generate_input_fn(model_name, ctx, mode):
    model = ctx.models[model_name]

    filenames = ctx.get_training_data_parts(model_name, mode)
    filenames = [aws.s3_path(ctx.bucket, f) for f in filenames]

    num_threads = multiprocessing.cpu_count()
    buffer_size = 2 * model[mode]["batch_size"] + 1

    def _input_fn():
        dataset = tf.data.TFRecordDataset(filenames=filenames)
        dataset = dataset.map(
            generate_example_parsing_fn(model_name, ctx, training=True),
            num_parallel_calls=num_threads,
        )

        dataset = dataset.prefetch(buffer_size)

        if model[mode]["shuffle"]:
            dataset = dataset.shuffle(buffer_size)

        dataset = dataset.repeat().batch(model[mode]["batch_size"])
        iterator = dataset.make_one_shot_iterator()
        features, target = iterator.get_next()

        return features, target

    return _input_fn


def generate_json_serving_input_fn(model_name, ctx):
    def _json_serving_input_fn():
        inputs = get_input_placeholder(model_name, ctx, training=False)
        features = {key: tf.expand_dims(tensor, -1) for key, tensor in inputs.items()}
        return tf.estimator.export.ServingInputReceiver(features=features, receiver_tensors=inputs)

    return _json_serving_input_fn


def generate_example_serving_input_fn(model_name, ctx):
    def _example_serving_input_fn():
        feature_spec = tf_lib.get_feature_spec(model_name, ctx, training=False)
        example_bytestring = tf.placeholder(shape=[None], dtype=tf.string)
        feature_scalars = tf.parse_single_example(example_bytestring, feature_spec)
        features = {key: tf.expand_dims(tensor, -1) for key, tensor in feature_scalars.items()}

        return tf.estimator.export.ServingInputReceiver(
            features=features, receiver_tensors={"example_proto": example_bytestring}
        )

    return _example_serving_input_fn


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

    tf_lib.set_logging_verbosity(model["misc"]["verbosity"])

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

    train_input_fn = generate_input_fn(model_name, ctx, "training")
    eval_input_fn = generate_input_fn(model_name, ctx, "evaluation")
    serving_input_fn = generate_json_serving_input_fn(model_name, ctx)
    exporter = tf.estimator.FinalExporter("estimator", serving_input_fn, as_text=False)

    dataset_metadata = aws.read_json_from_s3(model["dataset"]["metadata_key"], ctx.bucket)
    num_steps = model["training"]["num_steps"]
    if model["training"]["num_epochs"]:
        num_steps = (
            math.ceil(dataset_metadata["dataset_size"] / float(model["training"]["batch_size"]))
            * model["training"]["num_epochs"]
        )

    train_spec = tf.estimator.TrainSpec(train_input_fn, max_steps=num_steps)

    eval_spec = tf.estimator.EvalSpec(
        eval_input_fn,
        steps=model["evaluation"]["num_steps"],
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
