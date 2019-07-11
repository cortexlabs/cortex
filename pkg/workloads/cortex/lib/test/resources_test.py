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

import pytest

from cortex.lib.resources import ResourceMap


def test_resource_map_empty_resource():
    assert ResourceMap({}) == {}


def test_resource_map_single_resource():
    resource_map = {"a": {"name": "a", "id": "1"}}

    id_map = ResourceMap(resource_map)
    name_map = id_map.name_map
    assert id_map == {"1": {"name": "a", "id": "1", "aliases": ["a"]}}
    assert name_map == {"a": {"name": "a", "id": "1", "aliases": ["a"]}}


def test_resource_map_multiple_resource():

    resource_map = {
        "b": {"name": "b", "id": "1"},
        "a": {"name": "a", "id": "1"},
        "d": {"name": "d", "id": "1"},
        "c": {"name": "c", "id": "2"},
    }

    id_map = ResourceMap(resource_map)
    name_map = id_map.name_map
    assert id_map == {
        "1": {"name": "a", "id": "1", "aliases": ["a", "b", "d"]},
        "2": {"name": "c", "id": "2", "aliases": ["c"]},
    }
    assert name_map == {
        "a": {"name": "a", "id": "1", "aliases": ["a", "b", "d"]},
        "b": {"name": "b", "id": "1", "aliases": ["a", "b", "d"]},
        "d": {"name": "d", "id": "1", "aliases": ["a", "b", "d"]},
        "c": {"name": "c", "id": "2", "aliases": ["c"]},
    }


def test_ids_to_names():

    resource_map = {
        "b": {"name": "b", "id": "1"},
        "a": {"name": "a", "id": "1"},
        "d": {"name": "d", "id": "1"},
        "c": {"name": "c", "id": "2"},
    }

    id_map = ResourceMap(resource_map)
    assert sorted(id_map.ids_to_names("1")) == sorted(["a", "b", "d"])
    assert sorted(id_map.ids_to_names("2")) == sorted(["c"])
    assert sorted(id_map.ids_to_names("1", "2")) == sorted(["a", "b", "c", "d"])
