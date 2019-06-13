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

CORTEX_VERSION = "0.4.0"

COLUMN_TYPE_INT = "INT_COLUMN"
COLUMN_TYPE_FLOAT = "FLOAT_COLUMN"
COLUMN_TYPE_STRING = "STRING_COLUMN"
COLUMN_TYPE_INT_LIST = "INT_LIST_COLUMN"
COLUMN_TYPE_FLOAT_LIST = "FLOAT_LIST_COLUMN"
COLUMN_TYPE_STRING_LIST = "STRING_LIST_COLUMN"
COLUMN_TYPE_INFERRED = "INFERRED_COLUMN"

COLUMN_LIST_TYPES = [COLUMN_TYPE_INT_LIST, COLUMN_TYPE_FLOAT_LIST, COLUMN_TYPE_STRING_LIST]

COLUMN_TYPES = [
    COLUMN_TYPE_INT,
    COLUMN_TYPE_FLOAT,
    COLUMN_TYPE_STRING,
    COLUMN_TYPE_INT_LIST,
    COLUMN_TYPE_FLOAT_LIST,
    COLUMN_TYPE_STRING_LIST,
    COLUMN_TYPE_INFERRED,
]

VALUE_TYPE_INT = "INT"
VALUE_TYPE_FLOAT = "FLOAT"
VALUE_TYPE_STRING = "STRING"
VALUE_TYPE_BOOL = "BOOL"

VALUE_TYPES = [VALUE_TYPE_INT, VALUE_TYPE_FLOAT, VALUE_TYPE_STRING, VALUE_TYPE_BOOL]

ALL_TYPES = set(COLUMN_TYPES + VALUE_TYPES)
