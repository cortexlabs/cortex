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

import sys
import os

from lib import context


def test_create_inputs_from_features_map():
    features_values_map = {
        "e11": "value_11",
        "e12": 12,
        "e21": 2.1,
        "e22": "value_22",
        "e0": "value_e0",
        "e1": "value_e1",
        "e2": "value_e2",
        "e3": "value_e3",
        "e4": "value_e4",
    }

    feature_input_config = {"in": "e11"}
    inputs = context.create_inputs_from_features_map(features_values_map, feature_input_config)
    assert inputs == {"in": "value_11"}

    feature_input_config = {"a1": ["e11", "e12"], "a2": ["e21", "e22"]}
    inputs = context.create_inputs_from_features_map(features_values_map, feature_input_config)
    assert inputs == {"a1": ["value_11", 12], "a2": [2.1, "value_22"]}

    features_values_map2 = {
        "f1": 111,
        "f2": 2.22,
        "f3": "3",
        "f4": "4",
        "f5": "5",
        "f6": "6",
        "f7": "7",
        "f8": "8",
        "f9": "9",
    }

    feature_input_config = {"in1": "f1"}
    inputs = context.create_inputs_from_features_map(features_values_map2, feature_input_config)
    assert inputs == {"in1": 111}

    feature_input_config = {"in1": "f1", "in2": "f2"}
    inputs = context.create_inputs_from_features_map(features_values_map2, feature_input_config)
    assert inputs == {"in1": 111, "in2": 2.22}

    feature_input_config = {"in1": ["f1", "f2", "f3"]}
    inputs = context.create_inputs_from_features_map(features_values_map2, feature_input_config)
    assert inputs == {"in1": [111, 2.22, "3"]}

    feature_input_config = {"in1": ["f1", "f2", "f3"], "in2": ["f4", "f5", "f6"]}
    inputs = context.create_inputs_from_features_map(features_values_map2, feature_input_config)
    assert inputs == {"in1": [111, 2.22, "3"], "in2": ["4", "5", "6"]}
