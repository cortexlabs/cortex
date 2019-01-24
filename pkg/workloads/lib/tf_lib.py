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
import tensorflow as tf

from lib import util
import consts


CORTEX_TYPE_TO_TF_TYPE = {
    consts.FEATURE_TYPE_INT: tf.int64,
    consts.FEATURE_TYPE_INT_LIST: tf.int64,
    consts.FEATURE_TYPE_FLOAT: tf.float32,
    consts.FEATURE_TYPE_FLOAT_LIST: tf.float32,
    consts.FEATURE_TYPE_STRING: tf.string,
    consts.FEATURE_TYPE_STRING_LIST: tf.string,
}


def add_tf_types(config):
    if not util.is_dict(config):
        return

    type_fields = {}
    for k, v in config.items():
        if util.is_str(k) and util.is_str(v) and v in consts.FEATURE_TYPES:
            type_fields[k] = v
        elif util.is_dict(v):
            add_tf_types(v)
        elif util.is_list(v):
            for sub_v in v:
                add_tf_types(sub_v)

    for k, v in type_fields.items():
        config[k + "_tf"] = CORTEX_TYPE_TO_TF_TYPE[v]


def set_logging_verbosity(verbosity):
    tf.logging.set_verbosity(verbosity)
    os.environ["TF_CPP_MIN_LOG_LEVEL"] = str(tf.logging.__dict__[verbosity] / 10)


def get_feature_tf_types(model_name, ctx, training=True):
    """Generate a dict {feature name -> tf_type}"""
    model = ctx.models[model_name]

    feature_types = {
        feature_name: CORTEX_TYPE_TO_TF_TYPE[ctx.features[feature_name]["type"]]
        for feature_name in model["features"]
    }

    if training:
        target_feature_name = model["target"]
        feature_types[target_feature_name] = CORTEX_TYPE_TO_TF_TYPE[
            ctx.features[target_feature_name]["type"]
        ]

        for feature_name in model["training_features"]:
            feature_types[feature_name] = CORTEX_TYPE_TO_TF_TYPE[ctx.features[feature_name]["type"]]

    return feature_types


def get_feature_spec(model_name, ctx, training=True):
    """Generate a dict {feature name -> FixedLenFeatures} for use in tf.parse_example"""
    feature_types = get_feature_tf_types(model_name, ctx, training)
    feature_spec = {}
    for feature_name, tf_type in feature_types.items():
        if ctx.features[feature_name]["type"] in consts.FEATURE_LIST_TYPES:
            feature_spec[feature_name] = tf.FixedLenSequenceFeature(
                shape=(), dtype=tf_type, allow_missing=True
            )
        else:
            feature_spec[feature_name] = tf.FixedLenFeature(shape=(), dtype=tf_type)
    return feature_spec


def get_base_input_features(model_name, ctx):
    model = ctx.models[model_name]
    base_feature_names = set()
    for feature_name in model["features"]:
        if ctx.is_raw_feature(feature_name):
            base_feature_names.add(feature_name)
        else:
            transformed_feature = ctx.transformed_features[feature_name]
            for name in util.flatten_all_values(transformed_feature["inputs"]["features"]):
                base_feature_names.add(name)

    return [ctx.raw_features[name] for name in base_feature_names]
