# Copyright 2021 Cortex Labs, Inc.
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

from copy import deepcopy

from cortex_internal.lib import util


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
