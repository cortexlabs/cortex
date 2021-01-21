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

from collections.abc import Iterable


def truncate(item, max_elements=10, max_str_len=500):
    if isinstance(item, str):
        s = item
        if max_str_len > 3 and len(s) > max_str_len:
            s = s[: max_str_len - 3] + "..."
        return '"{}"'.format(s)

    if isinstance(item, dict):
        count = 0
        item_strs = []
        for key in item:
            if max_elements > 0 and count >= max_elements:
                item_strs.append("...")
                break

            key_str = truncate(key, max_elements, max_str_len)
            val_str = truncate(item[key], max_elements, max_str_len)
            item_strs.append("{}: {}".format(key_str, val_str))
            count += 1

        return "{" + ", ".join(item_strs) + "}"

    if isinstance(item, Iterable) and hasattr(item, "__getitem__"):
        item_strs = []

        for element in item[:max_elements]:
            item_strs.append(truncate(element, max_elements, max_str_len))

        if max_elements > 0 and len(item) > max_elements:
            item_strs.append("...")

        return "[" + ", ".join(item_strs) + "]"

    # Fallback
    s = str(item)
    if max_str_len > 3 and len(s) > max_str_len:
        s = s[: max_str_len - 3] + "..."
    return s
