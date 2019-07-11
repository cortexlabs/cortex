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

import collections
from copy import deepcopy

from cortex.lib import util


class ResourceMap(dict):
    def __init__(self, resource_name_map):
        dict.__init__(self)

        multi_map_by_id = {}
        name_map = {}
        multi_map = util.create_multi_map(resource_name_map, lambda k, v: v["id"])
        for k, resources in multi_map.items():
            sorted_resource_list = sorted(resources, key=lambda r: r["name"])
            aliases = [r["name"] for r in sorted_resource_list]

            for idx, resource in enumerate(sorted_resource_list):
                alias_resource = deepcopy(resource)
                alias_resource["aliases"] = aliases
                name_map[alias_resource["name"]] = alias_resource

                # use the lexicographically first resource (by name) as the representative resource in the id map
                if idx == 0:
                    multi_map_by_id[k] = alias_resource

        self.update(multi_map_by_id)
        self.name_map = name_map

    def ids_to_names(self, *resource_ids):
        list_of_list_of_names = (self[r]["aliases"] for r in resource_ids)
        return [name for name_list in list_of_list_of_names for name in name_list]
