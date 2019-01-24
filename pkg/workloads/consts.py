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

CORTEX_VERSION = "master"

FEATURE_TYPE_INT = "INT_FEATURE"
FEATURE_TYPE_FLOAT = "FLOAT_FEATURE"
FEATURE_TYPE_STRING = "STRING_FEATURE"
FEATURE_TYPE_INT_LIST = "INT_LIST_FEATURE"
FEATURE_TYPE_FLOAT_LIST = "FLOAT_LIST_FEATURE"
FEATURE_TYPE_STRING_LIST = "STRING_LIST_FEATURE"

FEATURE_LIST_TYPES = [FEATURE_TYPE_INT_LIST, FEATURE_TYPE_FLOAT_LIST, FEATURE_TYPE_STRING_LIST]

FEATURE_TYPES = [
    FEATURE_TYPE_INT,
    FEATURE_TYPE_FLOAT,
    FEATURE_TYPE_STRING,
    FEATURE_TYPE_INT_LIST,
    FEATURE_TYPE_FLOAT_LIST,
    FEATURE_TYPE_STRING_LIST,
]

VALUE_TYPE_INT = "INT"
VALUE_TYPE_FLOAT = "FLOAT"
VALUE_TYPE_STRING = "STRING"
VALUE_TYPE_BOOL = "BOOL"

VALUE_TYPES = [VALUE_TYPE_INT, VALUE_TYPE_FLOAT, VALUE_TYPE_STRING, VALUE_TYPE_BOOL]
