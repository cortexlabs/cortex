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

from lib import util
import logging


def test_snake_to_camel():
    assert util.snake_to_camel("ONE_TWO_THREE") == "oneTwoThree"
    assert util.snake_to_camel("ONE_TWO_THREE", lower=False) == "OneTwoThree"
    assert util.snake_to_camel("ONE_TWO_THREE", sep="-") == "one_two_three"
    assert util.snake_to_camel("ONE-TWO-THREE", sep="-") == "oneTwoThree"
    assert util.snake_to_camel("ONE") == "one"
    assert util.snake_to_camel("ONE", lower=False) == "One"


def test_flatten_all_values():
    obj = "v"
    expected = ["v"]
    assert util.flatten_all_values(obj) == expected

    obj = ["v1", "v2", "v3"]
    expected = ["v1", "v2", "v3"]
    assert util.flatten_all_values(obj) == expected

    obj = {"key1": "v1", "key2": "v2"}
    expected = ["v1", "v2"]
    assert util.flatten_all_values(obj) == expected

    obj = {
        "arr": ["v4", "v5", "v6"],
        "map": {"arr2": ["v7", "v8", "v9"]},
        "amap": {"arr2": ["v1", "v2", "v3"]},
        "zarr": [{"map": {"key1": "v10", "key2": "v11"}}],
    }
    expected = ["v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8", "v9", "v10", "v11"]
    assert util.flatten_all_values(obj) == expected

    obj = {
        "a1": ["TYPE1"],
        "m1": [
            {"mm1": "TYPE2", "mm2": "TYPE3", "mm3": ["TYPE4"], "mm4": ["TYPE5"]},
            ["TYPE6"],
            "TYPE7",
        ],
        "z1": "TYPE8",
    }
    expected = ["TYPE1", "TYPE2", "TYPE3", "TYPE4", "TYPE5", "TYPE6", "TYPE7", "TYPE8"]
    assert util.flatten_all_values(obj) == expected


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


def test_validate_column_type():
    assert util.validate_column_type(2, "INT_COLUMN") == True
    assert util.validate_column_type(2.2, "INT_COLUMN") == False
    assert util.validate_column_type("2", "INT_COLUMN") == False
    assert util.validate_column_type(None, "INT_COLUMN") == True

    assert util.validate_column_type(2.2, "FLOAT_COLUMN") == True
    assert util.validate_column_type(2, "FLOAT_COLUMN") == False
    assert util.validate_column_type("2", "FLOAT_COLUMN") == False
    assert util.validate_column_type(None, "FLOAT_COLUMN") == True

    assert util.validate_column_type("2", "STRING_COLUMN") == True
    assert util.validate_column_type(2, "STRING_COLUMN") == False
    assert util.validate_column_type(2.2, "STRING_COLUMN") == False
    assert util.validate_column_type(None, "STRING_COLUMN") == True

    assert util.validate_column_type("2", "STRING_LIST_COLUMN") == False
    assert util.validate_column_type(["2", "string"], "STRING_LIST_COLUMN") == True


def test_validate_output_type():
    assert util.validate_output_type(2, "INT") == True
    assert util.validate_output_type(2.2, "INT") == False
    assert util.validate_output_type("2", "INT") == False
    assert util.validate_output_type(None, "INT") == True

    assert util.validate_output_type(2.2, "FLOAT") == True
    assert util.validate_output_type(2, "FLOAT") == False
    assert util.validate_output_type("2", "FLOAT") == False
    assert util.validate_output_type(None, "FLOAT") == True

    assert util.validate_output_type(False, "BOOL") == True
    assert util.validate_output_type(2, "BOOL") == False
    assert util.validate_output_type("2", "BOOL") == False
    assert util.validate_output_type(None, "BOOL") == True

    assert util.validate_output_type(2.2, "INT|FLOAT") == True
    assert util.validate_output_type(2, "FLOAT|INT") == True
    assert util.validate_output_type("2", "FLOAT|INT") == False
    assert util.validate_output_type(None, "INT|FLOAT") == True

    assert util.validate_output_type({"test": 2.2}, {"STRING": "FLOAT"}) == True
    assert util.validate_output_type({"test": 2.2, "test2": 3.3}, {"STRING": "FLOAT"}) == True
    assert util.validate_output_type({}, {"STRING": "FLOAT"}) == True
    assert util.validate_output_type({"test": "2.2"}, {"STRING": "FLOAT"}) == False
    assert util.validate_output_type({2: 2.2}, {"STRING": "FLOAT"}) == False

    assert util.validate_output_type({"test": 2.2}, {"STRING": "INT|FLOAT"}) == True
    assert util.validate_output_type({"a": 2.2, "b": False}, {"STRING": "FLOAT|BOOL"}) == True
    assert util.validate_output_type({"test": 2.2, "test2": 3}, {"STRING": "FLOAT|BOOL"}) == False
    assert util.validate_output_type({}, {"STRING": "INT|FLOAT"}) == True
    assert util.validate_output_type({"test": "2.2"}, {"STRING": "FLOAT|INT"}) == False
    assert util.validate_output_type({2: 2.2}, {"STRING": "INT|FLOAT"}) == False

    assert util.validate_output_type({"f": 2.2, "i": 2}, {"f": "FLOAT", "i": "INT"}) == True
    assert util.validate_output_type({"f": 2.2, "i": 2.2}, {"f": "FLOAT", "i": "INT"}) == False
    assert util.validate_output_type({"f": "s", "i": 2}, {"f": "FLOAT", "i": "INT"}) == False
    assert util.validate_output_type({"f": 2.2}, {"f": "FLOAT", "i": "INT"}) == False
    assert util.validate_output_type({"f": 2.2, "i": None}, {"f": "FLOAT", "i": "INT"}) == True
    assert (
        util.validate_output_type({"f": 0.2, "i": 2, "e": 1}, {"f": "FLOAT", "i": "INT"}) == False
    )

    assert util.validate_output_type(["s"], ["STRING"]) == True
    assert util.validate_output_type(["a", "b", "c"], ["STRING"]) == True
    assert util.validate_output_type([], ["STRING"]) == True
    assert util.validate_output_type(None, ["STRING"]) == True
    assert util.validate_output_type([2], ["STRING"]) == False
    assert util.validate_output_type(["a", False, "c"], ["STRING"]) == False
    assert util.validate_output_type("a", ["STRING"]) == False

    assert util.validate_output_type([2], ["FLOAT|INT|BOOL"]) == True
    assert util.validate_output_type([2.2], ["FLOAT|INT|BOOL"]) == True
    assert util.validate_output_type([False], ["FLOAT|INT|BOOL"]) == True
    assert util.validate_output_type([2, 2.2, False], ["FLOAT|INT|BOOL"]) == True
    assert util.validate_output_type([], ["FLOAT|INT|BOOL"]) == True
    assert util.validate_output_type(None, ["FLOAT|INT|BOOL"]) == True
    assert util.validate_output_type([2, "s", True], ["FLOAT|INT|BOOL"]) == False
    assert util.validate_output_type(["s"], ["FLOAT|INT|BOOL"]) == False
    assert util.validate_output_type(2, ["FLOAT|INT|BOOL"]) == False

    value_type = {
        "map": {"STRING": "FLOAT"},
        "str": "STRING",
        "floats": ["FLOAT"],
        "map2": {
            "STRING": {
                "lat": "FLOAT",
                "lon": {
                    "a": "INT",
                    "b": ["STRING"],
                    "c": {"mean": "FLOAT", "sum": ["INT"], "stddev": {"STRING": "INT"}},
                },
                "bools": ["BOOL"],
                "anything": ["BOOL|INT|FLOAT|STRING"],
            }
        },
    }

    value = {
        "map": {"a": 2.2, "b": float(3)},
        "str": "test1",
        "floats": [2.2, 3.3, 4.4],
        "map2": {
            "testA": {
                "lat": 9.9,
                "lon": {
                    "a": 17,
                    "b": ["test1", "test2", "test3"],
                    "c": {"mean": 8.8, "sum": [3, 2, 1], "stddev": {"a": 1, "b": 2}},
                },
                "bools": [True],
                "anything": [],
            },
            "testB": {
                "lat": 3.14,
                "lon": {
                    "a": 88,
                    "b": ["testX", "testY", "testZ"],
                    "c": {"mean": 1.7, "sum": [1], "stddev": {"z": 12}},
                },
                "bools": [True, False, True],
                "anything": [10, 2.2, "test", False],
            },
        },
    }
    assert util.validate_output_type(value, value_type) == True

    value = {
        "map": {"a": 2.2, "b": float(3)},
        "str": "test1",
        "floats": [2.2, 3.3, 4.4],
        "map2": {
            "testA": {
                "lat": 9.9,
                "lon": {"a": 17, "b": ["test1", "test2", "test3"], "c": None},
                "bools": [True],
                "anything": [],
            },
            "testB": {
                "lat": None,
                "lon": {"a": 88, "b": None, "c": {"mean": 1.7, "sum": [1], "stddev": {"z": 12}}},
                "bools": [True, False, True],
                "anything": [10, 2.2, "test", False],
            },
            "testB": None,
        },
    }
    assert util.validate_output_type(value, value_type) == True

    value = {
        "map": {"a": 2.2, "b": float(3)},
        "str": "test1",
        "floats": [2.2, 3.3, 4.4],
        "map2": {
            "testA": {
                "lat": 9.9,
                "lon": {
                    "a": 17,
                    "b": ["test1", "test2", "test3"],
                    "c": {"mean": 8.8, "sum": [3, 2, 1], "stddev": {"a": 1, "b": 2}},
                },
                "bools": [True],
                "anything": [],
            },
            "testB": {
                "lat": 3.14,
                "lon": {
                    "b": ["testX", "testY", "testZ"],
                    "c": {"mean": 1.7, "sum": [1], "stddev": {"z": 12}},
                },
                "bools": [True, False, True],
                "anything": [10, 2.2, "test", False],
            },
        },
    }
    assert util.validate_output_type(value, value_type) == False

    value = {
        "map": {"a": 2.2, "b": float(3)},
        "str": "test1",
        "floats": [2.2, 3.3, 4.4],
        "map2": {
            "testA": {
                "lat": 9.9,
                "lon": {
                    "a": 17,
                    "b": ["test1", "test2", "test3"],
                    "c": {"mean": 8.8, "sum": [3, 2, 1], "stddev": {"a": 1, "b": 2}},
                },
                "bools": [True],
                "anything": [],
            },
            "testB": {
                "lat": 3.14,
                "lon": {
                    "a": 88.8,
                    "b": ["testX", "testY", "testZ"],
                    "c": {"mean": 1.7, "sum": [1], "stddev": {"z": 12}},
                },
                "bools": [True, False, True],
                "anything": [10, 2.2, "test", False],
            },
        },
    }
    assert util.validate_output_type(value, value_type) == False

    value = {
        "map": {"a": 2.2, "b": float(3)},
        "str": "test1",
        "floats": [2.2, 3.3, 4.4],
        "map2": {
            "testA": {
                "lat": 9.9,
                "lon": {
                    "a": 17,
                    "b": ["test1", "test2", "test3"],
                    "c": {"mean": 8.8, "sum": [3, 2, 1], "stddev": {"a": 1, "b": 2}},
                },
                "bools": [True],
                "anything": [],
            },
            "testB": {
                "lat": 3.14,
                "lon": {
                    "a": 88,
                    "b": ["testX", "testY", 2],
                    "c": {"mean": 1.7, "sum": [1], "stddev": {"z": 12}},
                },
                "bools": [True, False, True],
                "anything": [10, 2.2, "test", False],
            },
        },
    }
    assert util.validate_output_type(value, value_type) == False

    value = {
        "map": {"a": 2.2, "b": float(3)},
        "str": "test1",
        "floats": [2.2, 3.3, 4.4],
        "map2": {
            "testA": {
                "lat": 9.9,
                "lon": {
                    "a": 17,
                    "b": ["test1", "test2", "test3"],
                    "c": {"mean": 8.8, "sum": [3, 2, 1], "stddev": {"a": 1, "b": "test"}},
                },
                "bools": [True],
                "anything": [],
            },
            "testB": {
                "lat": 3.14,
                "lon": {
                    "a": 88,
                    "b": ["testX", "testY", "testZ"],
                    "c": {"mean": 1.7, "sum": [1], "stddev": {"z": 12}},
                },
                "bools": [True, False, True],
                "anything": [10, 2.2, "test", False],
            },
        },
    }
    assert util.validate_output_type(value, value_type) == False

    value = {
        "map": {"a": 2.2, "b": float(3)},
        "str": "test1",
        "floats": [2.2, 3.3, 4.4],
        "map2": {
            "testA": {
                "lat": 9.9,
                "lon": {
                    "a": 17,
                    "b": ["test1", "test2", "test3"],
                    "c": {"mean": 8.8, "sum": [3, 2, 1], "stddev": {"a": 1, "b": 2}},
                },
                "bools": [True],
                "anything": [],
            },
            "testB": {
                "lat": 3.14,
                "lon": {
                    "a": 88,
                    "b": ["testX", "testY", "testZ"],
                    "c": {"mean": 1.7, "sum": [1], "stddev": {"z": 12}},
                },
                "bools": True,
                "anything": [10, 2.2, "test", False],
            },
        },
    }
    assert util.validate_output_type(value, value_type) == False

    value = {
        "map": {"a": 2.2, "b": float(3)},
        "str": "test1",
        "floats": [2.2, 3.3, 4.4],
        "map2": {
            "testA": {
                "lat": 9.9,
                "lon": {
                    "a": 17,
                    "b": ["test1", "test2", "test3"],
                    "c": {"mean": 8.8, "sum": [3, 2, 1], "stddev": {"a": 1, "b": 2}},
                },
                "bools": [True],
                "anything": [],
            },
            "testB": {
                "lat": 3.14,
                "lon": {
                    "a": 88,
                    "b": ["testX", "testY", "testZ"],
                    "c": {"mean": 1.7, "sum": [1], "stddev": {"z": 12}},
                },
                "bools": [1, 2, 3],
                "anything": [10, 2.2, "test", False],
            },
        },
    }
    assert util.validate_output_type(value, value_type) == False
