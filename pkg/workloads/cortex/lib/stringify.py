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


def truncate(item, length=75):
    trim = length - 3

    if isinstance(item, str):
        data = "'{}".format(item[:trim])
        if length > 3 and len(item) > length:
            data += "..."
        data += "'"
        return data

    if isinstance(item, dict):
        s = "{"
        s += ",".join(["'{}': {}".format(key, truncate(item[key])) for key in item])
        s += "}"
        return s

    data = str(item).replace("\n", "")
    data = "{}...".format(data[:trim]) if length > 3 and len(data) > length else data
    if isinstance(item, list) or hasattr(item, "tolist"):
        data += "]"

    return data
