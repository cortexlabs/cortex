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
from copy import deepcopy
import pytest

from cortex.lib import util
import logging


def test_snake_to_camel():
    assert util.snake_to_camel("ONE_TWO_THREE") == "oneTwoThree"
    assert util.snake_to_camel("ONE_TWO_THREE", lower=False) == "OneTwoThree"
    assert util.snake_to_camel("ONE_TWO_THREE", sep="-") == "one_two_three"
    assert util.snake_to_camel("ONE-TWO-THREE", sep="-") == "oneTwoThree"
    assert util.snake_to_camel("ONE") == "one"
    assert util.snake_to_camel("ONE", lower=False) == "One"


def test_merge_dicts():
    dict1 = {"k1": "v1", "k2": "v2", "k3": {"k1": "v1", "k2": "v2"}}
    dict2 = {"k1": "V1", "k4": "V4", "k3": {"k1": "V1", "k4": "V4"}}

    expected1 = {"k1": "V1", "k2": "v2", "k4": "V4", "k3": {"k1": "V1", "k2": "v2", "k4": "V4"}}
    expected2 = {"k1": "v1", "k2": "v2", "k4": "V4", "k3": {"k1": "v1", "k2": "v2", "k4": "V4"}}

    merged = util.merge_dicts_overwrite(dict1, dict2)
    assert expected1 == merged
    assert dict1 != expected1
    assert dict2 != expected1

    merged = util.merge_dicts_no_overwrite(dict1, dict2)
    assert expected2 == merged
    assert dict1 != expected2
    assert dict2 != expected2

    dict1_copy = deepcopy(dict1)
    util.merge_dicts_in_place_overwrite(dict1_copy, dict2)
    assert expected1 == dict1_copy
    assert dict1 != dict1_copy

    dict1_copy = deepcopy(dict1)
    util.merge_dicts_in_place_no_overwrite(dict1_copy, dict2)
    assert expected2 == dict1_copy
    assert dict1 != dict1_copy


def test_is_number_col():
    assert util.is_number_col([1, 2, 3, 4])
    assert util.is_number_col([1, 0.2, 3, 0.4])
    assert not util.is_number_col([1, "2", 3, 0.4])
    assert not util.is_number_col(["1", "2", "3", ".4"])
    assert util.is_number_col([None, 1, None])
    assert util.is_number_col([None, 1.1, None])
    assert not util.is_number_col([None, None, None])


def test_print_samples_horiz(caplog):
    caplog.set_level(logging.INFO)

    samples = [
        {
            "test1": float(120),
            "test2": 0.2,
            "testlongstring": 0.20,
            "a": 0.23,
            "b": 0.233333333333333333,
        },
        {
            "test1": 11,
            "test2": 18,
            "testreallyreallyreallyreallyreallylongstring": 18,
            "a": -12,
            "b": 10,
        },
        {
            "test1": 13,
            "test2": 13,
            "testlongstring": 13,
            "testreallyreallyreallyreallyreallylongstring": 13,
            "a": 133333,
            "b": None,
        },
    ]
    util.print_samples_horiz(samples)

    records = [r.message for r in caplog.records]

    expected = (
        "a:                      0.23, -12, 133333\n"
        "b:                      0.23,  10,\n"
        "test1:                120.00,  11,     13\n"
        "test2:                  0.20,  18,     13\n"
        "testlongstring:         0.20,    ,     13\n"
        "testreallyreallyr...:       ,  18,     13\n"
    )

    assert "\n".join(records) + "\n" == expected
